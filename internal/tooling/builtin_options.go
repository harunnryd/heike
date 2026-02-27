package tooling

import (
	"fmt"
	"strings"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/tool"
)

func resolveBuiltinOptions(cfg *config.Config) (tool.BuiltinOptions, error) {
	if cfg == nil {
		return tool.BuiltinOptions{}, fmt.Errorf("config cannot be nil")
	}

	webTimeout, err := config.DurationOrDefault(cfg.Tools.Web.Timeout, config.DefaultWebToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.web.timeout: %w", err)
	}
	webBaseURL := strings.TrimSpace(cfg.Tools.Web.BaseURL)
	if webBaseURL == "" {
		webBaseURL = config.DefaultWebToolBaseURL
	}
	webMaxContentLength := cfg.Tools.Web.MaxContentLength
	if webMaxContentLength <= 0 {
		webMaxContentLength = config.DefaultWebToolMaxContentLength
	}

	weatherTimeout, err := config.DurationOrDefault(cfg.Tools.Weather.Timeout, config.DefaultWeatherToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.weather.timeout: %w", err)
	}
	weatherBaseURL := strings.TrimSpace(cfg.Tools.Weather.BaseURL)
	if weatherBaseURL == "" {
		weatherBaseURL = config.DefaultWeatherToolBaseURL
	}

	financeTimeout, err := config.DurationOrDefault(cfg.Tools.Finance.Timeout, config.DefaultFinanceToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.finance.timeout: %w", err)
	}
	financeBaseURL := strings.TrimSpace(cfg.Tools.Finance.BaseURL)
	if financeBaseURL == "" {
		financeBaseURL = config.DefaultFinanceToolBaseURL
	}

	sportsTimeout, err := config.DurationOrDefault(cfg.Tools.Sports.Timeout, config.DefaultSportsToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.sports.timeout: %w", err)
	}
	sportsBaseURL := strings.TrimSpace(cfg.Tools.Sports.BaseURL)
	if sportsBaseURL == "" {
		sportsBaseURL = config.DefaultSportsToolBaseURL
	}

	imageQueryTimeout, err := config.DurationOrDefault(cfg.Tools.ImageQuery.Timeout, config.DefaultImageQueryToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.image_query.timeout: %w", err)
	}
	imageQueryBaseURL := strings.TrimSpace(cfg.Tools.ImageQuery.BaseURL)
	if imageQueryBaseURL == "" {
		imageQueryBaseURL = config.DefaultImageQueryToolBaseURL
	}

	screenshotTimeout, err := config.DurationOrDefault(cfg.Tools.Screenshot.Timeout, config.DefaultScreenshotToolTimeout)
	if err != nil {
		return tool.BuiltinOptions{}, fmt.Errorf("parse tools.screenshot.timeout: %w", err)
	}
	screenshotRenderer := strings.TrimSpace(cfg.Tools.Screenshot.Renderer)
	if screenshotRenderer == "" {
		screenshotRenderer = config.DefaultScreenshotToolRenderer
	}

	applyPatchCommand := strings.TrimSpace(cfg.Tools.ApplyPatch.Command)
	if applyPatchCommand == "" {
		applyPatchCommand = config.DefaultApplyPatchToolCommand
	}

	return tool.BuiltinOptions{
		WebTimeout:          webTimeout,
		WebBaseURL:          webBaseURL,
		WebMaxContentLength: webMaxContentLength,
		WeatherBaseURL:      weatherBaseURL,
		WeatherTimeout:      weatherTimeout,
		FinanceBaseURL:      financeBaseURL,
		FinanceTimeout:      financeTimeout,
		SportsBaseURL:       sportsBaseURL,
		SportsTimeout:       sportsTimeout,
		ImageQueryBaseURL:   imageQueryBaseURL,
		ImageQueryTimeout:   imageQueryTimeout,
		ScreenshotTimeout:   screenshotTimeout,
		ScreenshotRenderer:  screenshotRenderer,
		ApplyPatchCommand:   applyPatchCommand,
	}, nil
}
