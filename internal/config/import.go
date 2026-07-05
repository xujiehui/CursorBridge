package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

type AdapterImportPreview struct {
	SourceType string         `json:"sourceType"`
	Adapters   []ModelAdapter `json:"adapters"`
	Warnings   []string       `json:"warnings"`
}

type importDefaults struct {
	ID          string
	DisplayName string
	Type        AdapterType
	BaseURL     string
	APIKey      string
	Enabled     bool
	HasEnabled  bool
}

type importModelSpec struct {
	ID          string
	DisplayName string
	Type        AdapterType
	BaseURL     string
	APIKey      string
	ModelID     string
	Enabled     bool
	HasEnabled  bool
}

type importBuildResult struct {
	Adapters []ModelAdapter
	Warnings []string
}

var errNoImportPayload = errors.New("no adapter import payload found")

func ParseAdapterImportSource(source string) (AdapterImportPreview, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return AdapterImportPreview{}, errors.New("import source is required")
	}

	if preview, err := parseImportURL(source); err == nil {
		return preview, nil
	}

	lastErr := errNoImportPayload
	for _, candidate := range decodeImportCandidates(source) {
		sourceType := "json"
		if candidate != source {
			sourceType = "base64-json"
		}
		preview, err := parseImportJSON(candidate, sourceType)
		if err == nil {
			return preview, nil
		}
		lastErr = err
	}
	if errors.Is(lastErr, errNoImportPayload) {
		return AdapterImportPreview{}, errors.New("could not find adapter import fields; expected baseURL, apiKey and model/models")
	}
	return AdapterImportPreview{}, lastErr
}

func parseImportURL(source string) (AdapterImportPreview, error) {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme == "" {
		return AdapterImportPreview{}, errNoImportPayload
	}

	query := parsed.Query()
	for _, name := range []string{"config", "payload", "data", "import", "adapter", "adapters"} {
		for _, value := range query[name] {
			for _, candidate := range decodeImportCandidates(value) {
				if preview, err := parseImportJSON(candidate, "link:"+name); err == nil {
					return preview, nil
				}
			}
		}
	}

	if hasQueryImportFields(query) {
		values := map[string]any{}
		for key, item := range query {
			if len(item) == 1 {
				values[key] = item[0]
				continue
			}
			parts := make([]any, 0, len(item))
			for _, value := range item {
				parts = append(parts, value)
			}
			values[key] = parts
		}
		result, err := buildAdaptersFromMap(values, importDefaults{})
		if err != nil {
			return AdapterImportPreview{}, err
		}
		return previewFromBuildResult("link:query", result)
	}

	pathPayload := strings.Trim(parsed.EscapedPath(), "/")
	if pathPayload == "" && parsed.Opaque != "" {
		pathPayload = parsed.Opaque
	}
	for _, value := range []string{pathPayload, parsed.Fragment} {
		for _, candidate := range decodeImportCandidates(value) {
			if preview, err := parseImportJSON(candidate, "link:payload"); err == nil {
				return preview, nil
			}
		}
	}

	return AdapterImportPreview{}, errNoImportPayload
}

func parseImportJSON(source string, sourceType string) (AdapterImportPreview, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return AdapterImportPreview{}, errNoImportPayload
	}
	var raw any
	if err := json.Unmarshal([]byte(source), &raw); err != nil {
		return AdapterImportPreview{}, err
	}
	result, err := buildAdaptersFromValue(raw, importDefaults{})
	if err != nil {
		return AdapterImportPreview{}, err
	}
	return previewFromBuildResult(sourceType, result)
}

func previewFromBuildResult(sourceType string, result importBuildResult) (AdapterImportPreview, error) {
	if len(result.Adapters) == 0 {
		return AdapterImportPreview{}, errors.New("import source did not include any model adapters")
	}
	ensureUniqueAdapterIDs(result.Adapters)
	return AdapterImportPreview{
		SourceType: sourceType,
		Adapters:   result.Adapters,
		Warnings:   result.Warnings,
	}, nil
}

func buildAdaptersFromValue(value any, defaults importDefaults) (importBuildResult, error) {
	switch typed := value.(type) {
	case []any:
		return buildAdaptersFromArray(typed, defaults)
	case map[string]any:
		return buildAdaptersFromMap(typed, defaults)
	default:
		return importBuildResult{}, errNoImportPayload
	}
}

func buildAdaptersFromArray(values []any, defaults importDefaults) (importBuildResult, error) {
	result := importBuildResult{}
	for index, value := range values {
		child, err := buildAdaptersFromValue(value, defaults)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ignored item %d: %s", index+1, err.Error()))
			continue
		}
		result.Adapters = append(result.Adapters, child.Adapters...)
		result.Warnings = append(result.Warnings, child.Warnings...)
	}
	if len(result.Adapters) == 0 {
		return importBuildResult{}, errors.New("no valid adapters found in import array")
	}
	return result, nil
}

func buildAdaptersFromMap(values map[string]any, defaults importDefaults) (importBuildResult, error) {
	nextDefaults := mergeImportDefaults(defaults, defaultsFromMap(values))
	result := importBuildResult{}

	for _, name := range []string{"modelAdapters", "adapters", "providers", "channels"} {
		if children, ok := getArray(values, name); ok {
			for index, childValue := range children {
				child, err := buildAdaptersFromValue(childValue, nextDefaults)
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("ignored %s item %d: %s", name, index+1, err.Error()))
					continue
				}
				result.Adapters = append(result.Adapters, child.Adapters...)
				result.Warnings = append(result.Warnings, child.Warnings...)
			}
		}
	}
	if len(result.Adapters) > 0 {
		return result, nil
	}

	models := modelSpecsFromMap(values)
	if len(models) == 0 {
		return importBuildResult{}, errors.New("adapter import requires model or models")
	}
	for _, model := range models {
		adapter := adapterFromModelSpec(model, nextDefaults, len(models) > 1)
		adapter = normalizeAdapter(adapter)
		if err := validateAdapter(adapter); err != nil {
			return importBuildResult{}, fmt.Errorf("invalid adapter for model %q: %w", model.ModelID, err)
		}
		result.Adapters = append(result.Adapters, adapter)
	}
	return result, nil
}

func defaultsFromMap(values map[string]any) importDefaults {
	defaults := importDefaults{
		Type:    AdapterOpenAI,
		Enabled: true,
	}
	defaults.ID = firstString(values, "id", "adapterID", "adapter_id", "providerID", "provider_id", "channelID", "channel_id")
	defaults.DisplayName = firstString(values, "displayName", "display_name", "name", "title", "providerName", "provider_name", "channelName", "channel_name")
	defaults.Type = importAdapterType(firstString(values, "type", "adapterType", "adapter_type", "provider", "service"))
	defaults.BaseURL = firstString(values, "baseURL", "base_url", "apiBase", "api_base", "apiBaseURL", "api_base_url", "apiURL", "api_url", "endpoint", "url")
	defaults.APIKey = firstString(values, "apiKey", "api_key", "key", "token", "accessToken", "access_token", "secret")
	if enabled, ok := firstBool(values, "enabled", "enable", "active"); ok {
		defaults.Enabled = enabled
		defaults.HasEnabled = true
	}
	return defaults
}

func mergeImportDefaults(parent, child importDefaults) importDefaults {
	merged := parent
	if child.ID != "" {
		merged.ID = child.ID
	}
	if child.DisplayName != "" {
		merged.DisplayName = child.DisplayName
	}
	if child.Type != "" {
		merged.Type = child.Type
	}
	if child.BaseURL != "" {
		merged.BaseURL = child.BaseURL
	}
	if child.APIKey != "" {
		merged.APIKey = child.APIKey
	}
	if child.HasEnabled {
		merged.Enabled = child.Enabled
		merged.HasEnabled = true
	}
	if merged.Type == "" {
		merged.Type = AdapterOpenAI
	}
	if !merged.HasEnabled {
		merged.Enabled = true
	}
	return merged
}

func modelSpecsFromMap(values map[string]any) []importModelSpec {
	if models, ok := getArray(values, "models", "modelIDs", "model_ids"); ok {
		specs := make([]importModelSpec, 0, len(models))
		for _, value := range models {
			if spec, ok := modelSpecFromValue(value); ok {
				specs = append(specs, spec)
			}
		}
		return specs
	}
	if models := splitList(firstString(values, "models", "modelIDs", "model_ids")); len(models) > 0 {
		specs := make([]importModelSpec, 0, len(models))
		for _, model := range models {
			specs = append(specs, importModelSpec{ModelID: model})
		}
		return specs
	}
	modelID := firstString(values, "modelID", "model_id", "model", "deployment", "deploymentID", "deployment_id")
	if modelID == "" {
		return nil
	}
	spec := importModelSpec{
		ID:          firstString(values, "id", "adapterID", "adapter_id"),
		DisplayName: firstString(values, "displayName", "display_name", "name", "title"),
		Type:        importAdapterType(firstString(values, "type", "adapterType", "adapter_type", "provider", "service")),
		BaseURL:     firstString(values, "baseURL", "base_url", "apiBase", "api_base", "apiBaseURL", "api_base_url", "apiURL", "api_url", "endpoint", "url"),
		APIKey:      firstString(values, "apiKey", "api_key", "key", "token", "accessToken", "access_token", "secret"),
		ModelID:     modelID,
	}
	if enabled, ok := firstBool(values, "enabled", "enable", "active"); ok {
		spec.Enabled = enabled
		spec.HasEnabled = true
	}
	return []importModelSpec{spec}
}

func modelSpecFromValue(value any) (importModelSpec, bool) {
	switch typed := value.(type) {
	case string:
		modelID := strings.TrimSpace(typed)
		if modelID == "" {
			return importModelSpec{}, false
		}
		return importModelSpec{ModelID: modelID}, true
	case map[string]any:
		modelID := firstString(typed, "modelID", "model_id", "model", "id", "name")
		if modelID == "" {
			return importModelSpec{}, false
		}
		spec := importModelSpec{
			ID:          firstString(typed, "adapterID", "adapter_id"),
			DisplayName: firstString(typed, "displayName", "display_name", "title", "name"),
			Type:        importAdapterType(firstString(typed, "type", "adapterType", "adapter_type", "provider", "service")),
			BaseURL:     firstString(typed, "baseURL", "base_url", "apiBase", "api_base", "apiBaseURL", "api_base_url", "apiURL", "api_url", "endpoint", "url"),
			APIKey:      firstString(typed, "apiKey", "api_key", "key", "token", "accessToken", "access_token", "secret"),
			ModelID:     modelID,
		}
		if enabled, ok := firstBool(typed, "enabled", "enable", "active"); ok {
			spec.Enabled = enabled
			spec.HasEnabled = true
		}
		return spec, true
	default:
		return importModelSpec{}, false
	}
}

func adapterFromModelSpec(model importModelSpec, defaults importDefaults, multiModel bool) ModelAdapter {
	adapterType := firstAdapterType(model.Type, defaults.Type, AdapterOpenAI)
	baseURL := firstNonEmpty(model.BaseURL, defaults.BaseURL)
	apiKey := firstNonEmpty(model.APIKey, defaults.APIKey)
	modelID := strings.TrimPrefix(strings.TrimSpace(model.ModelID), "byok/")
	displayName := firstNonEmpty(model.DisplayName, displayNameForModel(defaults.DisplayName, modelID))
	enabled := defaults.Enabled
	if model.HasEnabled {
		enabled = model.Enabled
	}

	id := model.ID
	if id == "" {
		if defaults.ID != "" && !multiModel {
			id = defaults.ID
		} else {
			id = adapterID(defaults.ID, defaults.DisplayName, baseURL, modelID)
		}
	}

	return ModelAdapter{
		ID:          id,
		DisplayName: displayName,
		Type:        adapterType,
		BaseURL:     baseURL,
		APIKey:      apiKey,
		ModelID:     modelID,
		Enabled:     enabled,
	}
}

func displayNameForModel(providerName, modelID string) string {
	providerName = strings.TrimSpace(providerName)
	modelID = strings.TrimSpace(modelID)
	if providerName == "" {
		return modelID
	}
	if modelID == "" {
		return providerName
	}
	return providerName + " " + modelID
}

func adapterID(prefix, displayName, baseURL, modelID string) string {
	parts := []string{prefix, displayName, hostForID(baseURL), modelID}
	joined := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if joined != "" {
			joined += "-"
		}
		joined += part
	}
	slug := slugID(joined)
	if slug == "" {
		return "adapter"
	}
	return slug
}

func ensureUniqueAdapterIDs(adapters []ModelAdapter) {
	seen := map[string]int{}
	for i := range adapters {
		base := slugID(adapters[i].ID)
		if base == "" {
			base = "adapter"
		}
		seen[base]++
		if seen[base] == 1 {
			adapters[i].ID = base
			continue
		}
		adapters[i].ID = fmt.Sprintf("%s-%d", base, seen[base])
	}
}

func slugID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, item := range value {
		isWord := item >= 'a' && item <= 'z' || item >= '0' && item <= '9'
		if isWord {
			builder.WriteRune(item)
			lastDash = false
			continue
		}
		if unicode.IsSpace(item) || item == '-' || item == '_' || item == '.' || item == '/' || item == ':' {
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func hostForID(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func importAdapterType(value string) AdapterType {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case value == "":
		return ""
	case strings.Contains(value, "anthropic") || strings.Contains(value, "claude"):
		return AdapterAnthropic
	default:
		return AdapterOpenAI
	}
}

func firstAdapterType(values ...AdapterType) AdapterType {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return AdapterOpenAI
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstString(values map[string]any, names ...string) string {
	for _, name := range names {
		value, ok := getValue(values, name)
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case json.Number:
			return typed.String()
		}
	}
	return ""
}

func firstBool(values map[string]any, names ...string) (bool, bool) {
	for _, name := range names {
		value, ok := getValue(values, name)
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "1", "true", "yes", "y", "on", "enabled":
				return true, true
			case "0", "false", "no", "n", "off", "disabled":
				return false, true
			}
		}
	}
	return false, false
}

func getArray(values map[string]any, names ...string) ([]any, bool) {
	for _, name := range names {
		value, ok := getValue(values, name)
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []any:
			return typed, true
		case []string:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				items = append(items, item)
			}
			return items, true
		case string:
			parts := splitList(typed)
			if len(parts) == 0 {
				continue
			}
			items := make([]any, 0, len(parts))
			for _, item := range parts {
				items = append(items, item)
			}
			return items, true
		}
	}
	return nil, false
}

func getValue(values map[string]any, name string) (any, bool) {
	target := importKey(name)
	for key, value := range values {
		if importKey(key) == target {
			return value, true
		}
	}
	return nil, false
}

func importKey(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, ".", "")
	return value
}

func splitList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func hasQueryImportFields(values url.Values) bool {
	required := [][]string{
		{"baseURL", "base_url", "apiBase", "api_base", "apiBaseURL", "api_base_url", "apiURL", "api_url", "endpoint", "url"},
		{"apiKey", "api_key", "key", "token", "accessToken", "access_token", "secret"},
		{"model", "modelID", "model_id", "models", "modelIDs", "model_ids"},
	}
	for _, group := range required {
		found := false
		for _, name := range group {
			if firstQuery(values, name) != "" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func firstQuery(values url.Values, name string) string {
	target := importKey(name)
	for key, items := range values {
		if importKey(key) != target {
			continue
		}
		for _, item := range items {
			if strings.TrimSpace(item) != "" {
				return strings.TrimSpace(item)
			}
		}
	}
	return ""
}

func decodeImportCandidates(source string) []string {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	if index := strings.Index(source, ","); strings.HasPrefix(strings.ToLower(source[:index+1]), "data:") {
		source = strings.TrimSpace(source[index+1:])
	}

	seen := map[string]bool{}
	candidates := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		candidates = append(candidates, value)
	}
	add(source)
	if unescaped, err := url.QueryUnescape(source); err == nil {
		add(unescaped)
	}
	for _, decoder := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if decoded, err := decoder.DecodeString(source); err == nil {
			add(string(decoded))
		}
	}
	return candidates
}
