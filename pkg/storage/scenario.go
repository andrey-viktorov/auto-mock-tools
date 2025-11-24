package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsonfilter "github.com/andrey-viktorov/jsonfilter-go"
	"github.com/andrey-viktorov/jsonfilter-go/serde"
	"gopkg.in/yaml.v3"
)

type scenarioFile struct {
	Scenarios []scenarioDefinition `yaml:"scenarios"`
}

type scenarioDefinition struct {
	Name     string                     `yaml:"name"`
	Method   string                     `yaml:"method"`
	Path     string                     `yaml:"path"`
	Filter   scenarioFilterDefinition   `yaml:"filter"`
	Response scenarioResponseDefinition `yaml:"response"`
}

type scenarioFilterDefinition struct {
	Body map[string]interface{} `yaml:"body"`
}

type scenarioResponseDefinition struct {
	File  string   `yaml:"file"`
	Delay *float64 `yaml:"delay"` // Optional override for response timing
}

type mockScenario struct {
	name        string
	path        string
	method      string
	methodBytes []byte
	filter      jsonfilter.Operator
	response    *MockResponse
}

// LoadScenarioConfig enables scenario-based matching using the supplied YAML file.
// When scenarios are present the legacy mock-id lookup path is disabled.
func (s *MockStorage) LoadScenarioConfig(configPath string) error {
	payload, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read scenario config: %w", err)
	}

	var file scenarioFile
	if err := yaml.Unmarshal(payload, &file); err != nil {
		return fmt.Errorf("parse scenario config: %w", err)
	}

	if len(file.Scenarios) == 0 {
		return fmt.Errorf("scenario config %s does not define any scenarios", configPath)
	}

	parser := serde.DefaultParser()
	baseDir := filepath.Dir(configPath)

	s.scenarioByPath = make(map[string][]*mockScenario)
	s.scenarioOrder = make([]*mockScenario, 0, len(file.Scenarios))

	for idx, def := range file.Scenarios {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			return fmt.Errorf("scenario #%d is missing name", idx+1)
		}

		path := strings.TrimSpace(def.Path)
		if path == "" {
			return fmt.Errorf("scenario %s is missing path", name)
		}

		responseFile := strings.TrimSpace(def.Response.File)
		if responseFile == "" {
			return fmt.Errorf("scenario %s is missing response.file", name)
		}

		resolvedFile := responseFile
		if !filepath.IsAbs(resolvedFile) {
			resolvedFile = filepath.Join(baseDir, resolvedFile)
		}

		mockResponse, err := loadResponseFromFile(resolvedFile, name)
		if err != nil {
			return fmt.Errorf("scenario %s: load response: %w", name, err)
		}

		// Apply delay override if specified
		if def.Response.Delay != nil {
			newDelay := *def.Response.Delay
			oldDelay := mockResponse.Delay

			// For SSE responses, redistribute timing across events proportionally
			if mockResponse.IsSSE && len(mockResponse.SSEEvents) > 0 && oldDelay > 0 {
				// Calculate scaling factor
				scale := newDelay / oldDelay

				// Rescale all event timestamps
				for i := range mockResponse.SSEEvents {
					mockResponse.SSEEvents[i].Timestamp *= scale
				}
			}

			mockResponse.Delay = newDelay
		}

		method := strings.ToUpper(strings.TrimSpace(def.Method))
		if method == "" {
			method = strings.ToUpper(mockResponse.Method)
		}
		if method == "" {
			method = "GET"
		}

		var operator jsonfilter.Operator
		if len(def.Filter.Body) > 0 {
			root := map[string]interface{}{"jsonFilter": def.Filter.Body}
			operator, err = parser.FromMap(root)
			if err != nil {
				return fmt.Errorf("scenario %s filter: %w", name, err)
			}

			validation := operator.Validate()
			if !validation.Valid {
				return fmt.Errorf("scenario %s filter invalid: %s", name, validation.CauseDescription)
			}
		}

		mockResponse.Path = path
		mockResponse.FullURL = path
		mockResponse.Method = method
		mockResponse.MethodBytes = []byte(method)
		mockResponse.MockID = name

		scenario := &mockScenario{
			name:        name,
			path:        path,
			method:      method,
			methodBytes: []byte(method),
			filter:      operator,
			response:    mockResponse,
		}

		s.scenarioByPath[path] = append(s.scenarioByPath[path], scenario)
		s.scenarioOrder = append(s.scenarioOrder, scenario)
	}

	s.scenariosEnabled = true
	// Refresh cached stats/list to reflect scenarios instead of legacy mock-id data.
	s.cacheResponses()

	return nil
}

// HasScenarios returns true when scenario-based routing is active.
func (s *MockStorage) HasScenarios() bool {
	return s.scenariosEnabled
}

// MatchScenarioResponse evaluates the configured scenarios in declaration order
// and returns the first response whose method and filter match.
func (s *MockStorage) MatchScenarioResponse(pathBytes, methodBytes, body []byte) *MockResponse {
	if !s.scenariosEnabled {
		return nil
	}

	scenarios := s.scenarioByPath[string(pathBytes)]
	if len(scenarios) == 0 {
		return nil
	}

	for _, scenario := range scenarios {
		if len(scenario.methodBytes) > 0 && len(methodBytes) > 0 && !equalFoldBytes(scenario.methodBytes, methodBytes) {
			continue
		}

		if scenario.filter != nil {
			result := scenario.filter.Evaluate(body)
			if !result.Match {
				continue
			}
		}

		return scenario.response
	}

	return nil
}
