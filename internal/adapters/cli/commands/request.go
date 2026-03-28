package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/adapters/cli/output"
	"github.com/ye-kart/reqflow/internal/app"
	coreauth "github.com/ye-kart/reqflow/internal/core/auth"
	"github.com/ye-kart/reqflow/internal/core/variable"
	"github.com/ye-kart/reqflow/internal/core/workflow"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/history"
)

// addBodyFlags adds --data and --content-type flags to commands that accept a body.
func addBodyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "request body (JSON string)")
	cmd.Flags().String("content-type", "", "content type")
}

// addAuthFlags adds authentication flags to a command.
func addAuthFlags(cmd *cobra.Command) {
	cmd.Flags().String("auth-basic", "", `basic auth credentials (format "username:password")`)
	cmd.Flags().String("auth-bearer", "", "bearer token")
	cmd.Flags().String("auth-apikey-header", "", `API key in header (format "HeaderName:Value")`)
	cmd.Flags().String("auth-apikey-query", "", `API key in query (format "paramName=Value")`)
	cmd.Flags().String("auth-oauth2-url", "", "OAuth2 token endpoint URL")
	cmd.Flags().String("auth-oauth2-client", "", `OAuth2 client credentials (format "client_id:client_secret")`)
	cmd.Flags().String("auth-digest", "", `digest auth credentials (format "username:password")`)
	cmd.Flags().String("auth-aws", "", `AWS credentials (format "access_key:secret_key")`)
	cmd.Flags().String("auth-aws-region", "", "AWS region for signature")
	cmd.Flags().String("auth-aws-service", "", "AWS service name for signature")
}

// addRetryFlags adds retry-related flags to a command.
func addRetryFlags(cmd *cobra.Command) {
	cmd.Flags().Int("retry", 0, "max number of retries")
	cmd.Flags().String("retry-backoff", "fixed", `backoff strategy: "fixed", "linear", "exponential"`)
	cmd.Flags().String("retry-on", "", "comma-separated HTTP status codes to retry on (e.g., 502,503,504)")
}

// addCurlFlag adds the --curl flag for exporting instead of executing.
func addCurlFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("curl", false, "print the equivalent cURL command instead of executing")
}

// addPollFlags adds polling flags to a command.
func addPollFlags(cmd *cobra.Command) {
	cmd.Flags().String("wait-for", "", `poll until JSONPath condition is met (e.g. "$.status == 'completed'")`)
	cmd.Flags().Duration("poll", 2*time.Second, "interval between poll attempts")
	cmd.Flags().Duration("poll-timeout", 60*time.Second, "maximum time to wait for condition")
}

// addFailOnErrorFlag adds the --fail-on-error and --no-fail-on-error flags.
func addFailOnErrorFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("fail-on-error", true, "exit with non-zero code on HTTP 4xx/5xx")
	cmd.Flags().Bool("no-fail-on-error", false, "do not exit with non-zero code on HTTP 4xx/5xx")
}

// addExtractFlag adds the --extract flag for inline value extraction.
func addExtractFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("extract", nil, `extract value from response (e.g. "$.field" or "name=$.field")`)
}

// addAssertFlag adds the --assert flag for inline response assertions.
func addAssertFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("assert", nil, `assert response condition (e.g. "status == 200")`)
}

// addCookieFlags adds cookie-related flags to a command.
func addCookieFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("no-cookies", false, "disable cookie handling for this request")
}

// addHistoryFlags adds history-related flags to a command.
func addHistoryFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("no-history", false, "do not save this request to history")
}

func newGetCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <url>",
		Short: "Send a GET request",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunE(a, domain.MethodGet, false),
	}
	addAuthFlags(cmd)
	addCurlFlag(cmd)
	addCodegenFlag(cmd)
	addFailOnErrorFlag(cmd)
	addPollFlags(cmd)
	addRetryFlags(cmd)
	addExtractFlag(cmd)
	addAssertFlag(cmd)
	addCookieFlags(cmd)
	addHistoryFlags(cmd)
	return cmd
}

func newPostCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post <url>",
		Short: "Send a POST request",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunE(a, domain.MethodPost, true),
	}
	addBodyFlags(cmd)
	addAuthFlags(cmd)
	addCurlFlag(cmd)
	addCodegenFlag(cmd)
	addFailOnErrorFlag(cmd)
	addPollFlags(cmd)
	addRetryFlags(cmd)
	addExtractFlag(cmd)
	addAssertFlag(cmd)
	addCookieFlags(cmd)
	addHistoryFlags(cmd)
	return cmd
}

func newPutCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <url>",
		Short: "Send a PUT request",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunE(a, domain.MethodPut, true),
	}
	addBodyFlags(cmd)
	addAuthFlags(cmd)
	addCurlFlag(cmd)
	addCodegenFlag(cmd)
	addFailOnErrorFlag(cmd)
	addPollFlags(cmd)
	addRetryFlags(cmd)
	addExtractFlag(cmd)
	addAssertFlag(cmd)
	addCookieFlags(cmd)
	addHistoryFlags(cmd)
	return cmd
}

func newPatchCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch <url>",
		Short: "Send a PATCH request",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunE(a, domain.MethodPatch, true),
	}
	addBodyFlags(cmd)
	addAuthFlags(cmd)
	addCurlFlag(cmd)
	addCodegenFlag(cmd)
	addFailOnErrorFlag(cmd)
	addPollFlags(cmd)
	addRetryFlags(cmd)
	addExtractFlag(cmd)
	addAssertFlag(cmd)
	addCookieFlags(cmd)
	addHistoryFlags(cmd)
	return cmd
}

func newDeleteCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <url>",
		Short: "Send a DELETE request",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunE(a, domain.MethodDelete, false),
	}
	addAuthFlags(cmd)
	addCurlFlag(cmd)
	addCodegenFlag(cmd)
	addFailOnErrorFlag(cmd)
	addPollFlags(cmd)
	addRetryFlags(cmd)
	addExtractFlag(cmd)
	addAssertFlag(cmd)
	addCookieFlags(cmd)
	addHistoryFlags(cmd)
	return cmd
}

// makeRunE creates a RunE function for an HTTP method command.
func makeRunE(a *app.App, method domain.HTTPMethod, hasBody bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		url := args[0]

		config := domain.RequestConfig{
			Method: method,
			URL:    url,
		}

		// Parse timeout from persistent flags.
		timeout, _ := cmd.Flags().GetDuration("timeout")
		if timeout > 0 {
			config.Timeout = timeout
		}

		// Build headers: config defaults first, then CLI flags override by key.
		cfg := configFromContext(cmd.Context())
		var configHeaders []domain.Header
		if cfg != nil {
			configHeaders = cfg.DefaultHeaders
		}
		headers, _ := cmd.Flags().GetStringSlice("header")
		var cliHeaders []domain.Header
		for _, h := range headers {
			key, value, ok := parseHeader(h)
			if ok {
				cliHeaders = append(cliHeaders, domain.Header{Key: key, Value: value})
			}
		}
		config.Headers = mergeHeaders(configHeaders, cliHeaders)

		// Parse query params from persistent flags.
		queries, _ := cmd.Flags().GetStringSlice("query")
		for _, q := range queries {
			key, value, ok := parseQueryParam(q)
			if ok {
				config.QueryParams = append(config.QueryParams, domain.QueryParam{Key: key, Value: value})
			}
		}

		// Parse body flags if applicable.
		if hasBody {
			data, _ := cmd.Flags().GetString("data")
			if data != "" {
				config.Body = []byte(data)
			}
			contentType, _ := cmd.Flags().GetString("content-type")
			if contentType != "" {
				config.ContentType = contentType
			}
		}

		// Parse auth flags.
		authConfig, err := parseAuthFlags(cmd)
		if err != nil {
			return err
		}
		if authConfig != nil {
			// OAuth2 requires fetching a token before applying auth.
			if authConfig.Type == domain.AuthOAuth2 && authConfig.OAuth2 != nil {
				tokenCtx, tokenCancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
				defer tokenCancel()
				token, tokenErr := coreauth.FetchOAuth2Token(tokenCtx, *authConfig.OAuth2, a.HTTPClient())
				if tokenErr != nil {
					return fmt.Errorf("oauth2 token fetch failed: %w", tokenErr)
				}
				// Replace with a Bearer auth config containing the fetched token.
				authConfig = &domain.AuthConfig{
					Type:   domain.AuthBearer,
					Bearer: &domain.BearerAuthConfig{Token: token},
				}
			}
			config.Auth = authConfig
		}

		// Parse cookie flags.
		noCookies, _ := cmd.Flags().GetBool("no-cookies")
		if noCookies {
			config.NoCookies = true
		}

		// If --curl flag is set, print the curl equivalent and return.
		curlExport, _ := cmd.Flags().GetBool("curl")
		if curlExport {
			return printCurlExport(cmd, config)
		}

		// If --codegen flag is set, generate code and return.
		codegenLang, _ := cmd.Flags().GetString("codegen")
		if codegenLang != "" {
			return printCodegen(cmd, codegenLang, config)
		}

		// Load environment variables if -e flag is set.
		var vars map[string]string
		envName, _ := cmd.Flags().GetString("env")
		if envName != "" && a.Storage != nil {
			envDir, _ := cmd.Flags().GetString("env-dir")
			envPath := filepath.Join(envDir, envName+".yaml")
			env, err := a.Storage.ReadEnvironment(envPath)
			if err != nil {
				return fmt.Errorf("loading environment %q: %w", envName, err)
			}
			vars = variable.Resolve(env.Variables)
		}

		// Parse display flags.
		verbose, _ := cmd.Flags().GetBool("verbose")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		trace, _ := cmd.Flags().GetBool("trace")
		noColor, _ := cmd.Flags().GetBool("no-color")
		w := cmd.OutOrStdout()

		// Dry-run mode: build request and display it without sending.
		if dryRun {
			req, err := a.HTTPExecutor.BuildRequest(config, nil)
			if err != nil {
				return fmt.Errorf("building request: %w", err)
			}
			return output.FormatDryRun(w, req, noColor)
		}

		// Enable trace timing if requested.
		if trace {
			a.EnableTrace()
		}

		// Check for polling flags.
		waitFor, _ := cmd.Flags().GetString("wait-for")
		if waitFor != "" {
			return executePollRequest(cmd, a, config, vars, waitFor, verbose, trace, noColor, timeout)
		}

		// Parse retry flags.
		retryCfg, err := parseRetryFlags(cmd)
		if err != nil {
			return err
		}

		// Execute the request (with retry if configured).
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
		defer cancel()

		var result domain.ExecutionResult
		if retryCfg != nil {
			result, err = executeWithRetry(ctx, a, config, vars, retryCfg)
		} else {
			result, err = a.HTTPExecutor.Execute(ctx, config, vars)
		}
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		// Save to history unless --no-history is set.
		noHistory, _ := cmd.Flags().GetBool("no-history")
		if !noHistory && a.HistoryStore != nil {
			entry := history.Entry{
				ID:        history.GenerateID(),
				Timestamp: time.Now(),
				Method:    string(method),
				URL:       url,
				Status:    result.Response.StatusCode,
				Duration:  result.Response.Duration,
				Request:   result.Request,
				Response:  result.Response,
			}
			// Best-effort: don't fail the command if history saving fails.
			_ = a.HistoryStore.Add(entry)
		}

		// Check for --extract and --assert flags.
		extractExprs, _ := cmd.Flags().GetStringSlice("extract")
		assertExprs, _ := cmd.Flags().GetStringSlice("assert")
		hasExtract := len(extractExprs) > 0
		hasAssert := len(assertExprs) > 0

		if hasExtract {
			// Extract mode: skip normal response formatting.
			if err := handleExtract(cmd, result.Response, extractExprs); err != nil {
				return err
			}
		} else {
			// Verbose mode: show request and response details.
			if verbose {
				if err := output.FormatVerbose(w, result.Request, result.Response, noColor); err != nil {
					return err
				}
			} else {
				// Normal mode: format and write response.
				outputFmt, _ := cmd.Flags().GetString("output")
				formatter := output.New(domain.OutputFormat(outputFmt), noColor)
				if err := formatter.FormatResponse(w, result.Response); err != nil {
					return err
				}
			}
		}

		// Assertions are evaluated AFTER extract.
		if hasAssert {
			if err := handleAssert(cmd, result.Response, assertExprs); err != nil {
				return err
			}
		}

		// Show trace timing if requested.
		if trace {
			fmt.Fprintln(w)
			if err := output.FormatTrace(w, result.Response.Timing, noColor); err != nil {
				return err
			}
		}

		// Check if response indicates an error and --fail-on-error is active.
		if result.Response.StatusCode >= 400 {
			if shouldFailOnError(cmd) {
				return domain.NewHTTPError(result.Response.StatusCode, nil)
			}
		}

		return nil
	}
}

// parseHeader splits a header string on the first ": " separator.
func parseHeader(s string) (key, value string, ok bool) {
	idx := strings.Index(s, ": ")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+2:], true
}

// parseQueryParam splits a query param string on the first "=" separator.
func parseQueryParam(s string) (key, value string, ok bool) {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

// parseAuthFlags reads auth-related flags and returns an AuthConfig if any are set.
func parseAuthFlags(cmd *cobra.Command) (*domain.AuthConfig, error) {
	basic, _ := cmd.Flags().GetString("auth-basic")
	if basic != "" {
		parts := strings.SplitN(basic, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-basic format, expected 'username:password'")
		}
		return &domain.AuthConfig{
			Type: domain.AuthBasic,
			Basic: &domain.BasicAuthConfig{
				Username: parts[0],
				Password: parts[1],
			},
		}, nil
	}

	bearer, _ := cmd.Flags().GetString("auth-bearer")
	if bearer != "" {
		return &domain.AuthConfig{
			Type: domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{
				Token: bearer,
			},
		}, nil
	}

	apikeyHeader, _ := cmd.Flags().GetString("auth-apikey-header")
	if apikeyHeader != "" {
		parts := strings.SplitN(apikeyHeader, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-apikey-header format, expected 'HeaderName:Value'")
		}
		return &domain.AuthConfig{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuthConfig{
				Key:      parts[0],
				Value:    parts[1],
				Location: domain.APIKeyInHeader,
			},
		}, nil
	}

	apikeyQuery, _ := cmd.Flags().GetString("auth-apikey-query")
	if apikeyQuery != "" {
		parts := strings.SplitN(apikeyQuery, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-apikey-query format, expected 'paramName=Value'")
		}
		return &domain.AuthConfig{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuthConfig{
				Key:      parts[0],
				Value:    parts[1],
				Location: domain.APIKeyInQuery,
			},
		}, nil
	}

	oauth2URL, _ := cmd.Flags().GetString("auth-oauth2-url")
	if oauth2URL != "" {
		clientCreds, _ := cmd.Flags().GetString("auth-oauth2-client")
		if clientCreds == "" {
			return nil, fmt.Errorf("--auth-oauth2-client is required when using --auth-oauth2-url")
		}
		parts := strings.SplitN(clientCreds, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-oauth2-client format, expected 'client_id:client_secret'")
		}
		return &domain.AuthConfig{
			Type: domain.AuthOAuth2,
			OAuth2: &domain.OAuth2Config{
				TokenURL:     oauth2URL,
				ClientID:     parts[0],
				ClientSecret: parts[1],
				GrantType:    "client_credentials",
			},
		}, nil
	}

	digest, _ := cmd.Flags().GetString("auth-digest")
	if digest != "" {
		parts := strings.SplitN(digest, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-digest format, expected 'username:password'")
		}
		return &domain.AuthConfig{
			Type: domain.AuthDigest,
			Digest: &domain.DigestAuthConfig{
				Username: parts[0],
				Password: parts[1],
			},
		}, nil
	}

	awsCreds, _ := cmd.Flags().GetString("auth-aws")
	if awsCreds != "" {
		parts := strings.SplitN(awsCreds, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --auth-aws format, expected 'access_key:secret_key'")
		}
		region, _ := cmd.Flags().GetString("auth-aws-region")
		service, _ := cmd.Flags().GetString("auth-aws-service")
		return &domain.AuthConfig{
			Type: domain.AuthAWS,
			AWS: &domain.AWSAuthConfig{
				AccessKey: parts[0],
				SecretKey: parts[1],
				Region:    region,
				Service:   service,
			},
		}, nil
	}

	return nil, nil
}

// mergeHeaders combines config default headers with CLI flag headers.
// CLI headers override config headers with the same key.
func mergeHeaders(configHeaders, cliHeaders []domain.Header) []domain.Header {
	if len(configHeaders) == 0 {
		return cliHeaders
	}

	// Build set of CLI header keys for fast lookup.
	cliKeys := make(map[string]bool, len(cliHeaders))
	for _, h := range cliHeaders {
		cliKeys[h.Key] = true
	}

	// Start with config headers that are not overridden.
	merged := make([]domain.Header, 0, len(configHeaders)+len(cliHeaders))
	for _, h := range configHeaders {
		if !cliKeys[h.Key] {
			merged = append(merged, h)
		}
	}
	// Append all CLI headers.
	merged = append(merged, cliHeaders...)
	return merged
}

// shouldFailOnError returns true if the command should return an error for HTTP 4xx/5xx.
// --no-fail-on-error takes precedence over --fail-on-error.
func shouldFailOnError(cmd *cobra.Command) bool {
	noFail, _ := cmd.Flags().GetBool("no-fail-on-error")
	if noFail {
		return false
	}
	failOnError, _ := cmd.Flags().GetBool("fail-on-error")
	return failOnError
}

// resolveTimeout returns a sensible timeout value.
func resolveTimeout(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return 30 * time.Second
}

// executePollRequest handles the --wait-for polling execution path.
func executePollRequest(cmd *cobra.Command, a *app.App, config domain.RequestConfig, vars map[string]string, waitFor string, verbose, trace, noColor bool, timeout time.Duration) error {
	pollInterval, _ := cmd.Flags().GetDuration("poll")
	pollTimeout, _ := cmd.Flags().GetDuration("poll-timeout")

	pollCtx, cancel := context.WithTimeout(context.Background(), pollTimeout)
	defer cancel()

	w := cmd.OutOrStdout()
	attempt := 0

	for {
		attempt++

		// Use per-request timeout if set, otherwise use a reasonable default
		reqCtx, reqCancel := context.WithTimeout(pollCtx, resolveTimeout(timeout))
		result, err := a.HTTPExecutor.Execute(reqCtx, config, vars)
		reqCancel()

		if err != nil {
			if pollCtx.Err() != nil {
				return fmt.Errorf("polling timed out after %d attempts", attempt)
			}
			return fmt.Errorf("poll request failed: %w", err)
		}

		// Check condition
		conditionMet, condErr := workflow.EvaluateCondition(result.Response.Body, waitFor)
		if condErr != nil {
			return fmt.Errorf("evaluating poll condition: %w", condErr)
		}

		if conditionMet {
			// Output the final successful response
			if verbose {
				if err := output.FormatVerbose(w, result.Request, result.Response, noColor); err != nil {
					return err
				}
			} else {
				outputFmt, _ := cmd.Flags().GetString("output")
				formatter := output.New(domain.OutputFormat(outputFmt), noColor)
				if err := formatter.FormatResponse(w, result.Response); err != nil {
					return err
				}
			}

			if trace {
				fmt.Fprintln(w)
				if err := output.FormatTrace(w, result.Response.Timing, noColor); err != nil {
					return err
				}
			}

			if result.Response.StatusCode >= 400 {
				if shouldFailOnError(cmd) {
					return domain.NewHTTPError(result.Response.StatusCode, nil)
				}
			}

			return nil
		}

		// Wait for the next poll interval or context cancellation
		select {
		case <-pollCtx.Done():
			return fmt.Errorf("polling timed out after %d attempts", attempt)
		case <-time.After(pollInterval):
			// Continue to next attempt
		}
	}
}

// parseRetryFlags reads retry-related flags and returns a RetryConfig if --retry is set.
func parseRetryFlags(cmd *cobra.Command) (*domain.RetryConfig, error) {
	retryMax, _ := cmd.Flags().GetInt("retry")
	if retryMax <= 0 {
		return nil, nil
	}

	backoff, _ := cmd.Flags().GetString("retry-backoff")
	retryOnStr, _ := cmd.Flags().GetString("retry-on")

	var retryOn []int
	if retryOnStr != "" {
		for _, s := range strings.Split(retryOnStr, ",") {
			s = strings.TrimSpace(s)
			code, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("invalid status code in --retry-on: %q", s)
			}
			retryOn = append(retryOn, code)
		}
	}

	return &domain.RetryConfig{
		Max:          retryMax,
		Backoff:      backoff,
		InitialDelay: 1 * time.Second,
		RetryOn:      retryOn,
		RetryOnError: true,
	}, nil
}

// executeWithRetry wraps the HTTP executor with retry logic for CLI commands.
func executeWithRetry(ctx context.Context, a *app.App, config domain.RequestConfig, vars map[string]string, cfg *domain.RetryConfig) (domain.ExecutionResult, error) {
	var lastResult domain.ExecutionResult
	var lastErr error

	for attempt := 0; attempt <= cfg.Max; attempt++ {
		if attempt > 0 {
			delay := workflow.CalculateDelay(cfg.Backoff, attempt, cfg.InitialDelay)
			if cfg.Jitter {
				delay = workflow.ApplyJitter(delay)
			}

			select {
			case <-ctx.Done():
				return lastResult, fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		lastResult, lastErr = a.HTTPExecutor.Execute(ctx, config, vars)

		// Network error: retry if configured
		if lastErr != nil {
			if cfg.RetryOnError && attempt < cfg.Max {
				continue
			}
			return lastResult, lastErr
		}

		// Check if status code is retryable
		if shouldRetryStatusCode(lastResult.Response.StatusCode, cfg.RetryOn) && attempt < cfg.Max {
			continue
		}

		return lastResult, nil
	}

	return lastResult, lastErr
}

// shouldRetryStatusCode checks if the given status code is in the retryable list.
func shouldRetryStatusCode(statusCode int, retryOn []int) bool {
	for _, code := range retryOn {
		if statusCode == code {
			return true
		}
	}
	return false
}

// handleExtract processes --extract flags, extracts values from the response
// body, and prints them. When --output json is set, output is a JSON object.
func handleExtract(cmd *cobra.Command, resp domain.HTTPResponse, exprs []string) error {
	w := cmd.OutOrStdout()

	// Build extraction map from expressions.
	extractMap := make(map[string]string)
	// Track insertion order for deterministic text output.
	type extractEntry struct {
		varName  string
		jsonPath string
	}
	var entries []extractEntry

	for _, expr := range exprs {
		varName, jsonPath, err := workflow.ParseExtractString(expr)
		if err != nil {
			return fmt.Errorf("invalid --extract %q: %w", expr, err)
		}
		// Use the jsonPath as key if no label is given.
		key := varName
		if key == "" {
			key = jsonPath
		}
		extractMap[key] = jsonPath
		entries = append(entries, extractEntry{varName: varName, jsonPath: jsonPath})
	}

	values, err := workflow.ExtractValues(resp.Body, extractMap)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Check if JSON output is requested.
	outputFmt, _ := cmd.Flags().GetString("output")
	if domain.OutputFormat(outputFmt) == domain.OutputJSON {
		// Build a clean map with labels as keys.
		jsonOut := make(map[string]string, len(entries))
		for _, e := range entries {
			key := e.varName
			if key == "" {
				key = e.jsonPath
			}
			jsonOut[key] = values[key]
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonOut)
	}

	// Text output: one value per line.
	for _, e := range entries {
		key := e.varName
		if key == "" {
			key = e.jsonPath
		}
		val := values[key]
		if e.varName != "" {
			fmt.Fprintf(w, "%s=%s\n", e.varName, val)
		} else {
			fmt.Fprintln(w, val)
		}
	}

	return nil
}

// handleAssert processes --assert flags, evaluates assertions against the
// response, and prints results. Returns an AssertionError if any fail.
func handleAssert(cmd *cobra.Command, resp domain.HTTPResponse, exprs []string) error {
	w := cmd.OutOrStdout()

	var assertions []domain.Assertion
	for _, expr := range exprs {
		a, err := workflow.ParseAssertionString(expr)
		if err != nil {
			return fmt.Errorf("invalid --assert %q: %w", expr, err)
		}
		assertions = append(assertions, a)
	}

	results := workflow.EvaluateAssertions(assertions, resp)

	anyFailed := false
	for _, r := range results {
		if r.Passed {
			fmt.Fprintf(w, "\u2713 %s %s %v\n", r.Assertion.Field, r.Assertion.Operator, r.Assertion.Expected)
		} else {
			anyFailed = true
			fmt.Fprintf(w, "\u2717 %s %s %v\n", r.Assertion.Field, r.Assertion.Operator, r.Assertion.Expected)
			if r.Message != "" {
				fmt.Fprintf(w, "  %s\n", r.Message)
			}
		}
	}

	if anyFailed {
		return domain.NewAssertionError("one or more assertions failed")
	}

	return nil
}
