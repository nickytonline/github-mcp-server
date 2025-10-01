package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/github-mcp-server/internal/ghmcp"
	"github.com/github/github-mcp-server/pkg/github"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// These variables are set by the build process using ldflags.
var version = "version"
var commit = "commit"
var date = "date"

var (
	rootCmd = &cobra.Command{
		Use:     "server",
		Short:   "GitHub MCP Server",
		Long:    `A GitHub MCP server that handles various tools and resources.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:    "stdio",
		Short:  "Start stdio server",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("stdio transport has been removed; run `github-mcp-server http` behind your OAuth proxy")
		},
	}

	httpCmd = &cobra.Command{
		Use:   "http",
		Short: "Start HTTP server",
		Long:  `Start a server that communicates over HTTP using the streamable transport.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			var enabledToolsets []string
			if err := viper.UnmarshalKey("toolsets", &enabledToolsets); err != nil {
				return fmt.Errorf("failed to unmarshal toolsets: %w", err)
			}

			httpServerConfig := ghmcp.HTTPServerConfig{
				Version:           version,
				Host:              viper.GetString("host"),
				EnabledToolsets:   enabledToolsets,
				DynamicToolsets:   viper.GetBool("dynamic_toolsets"),
				ReadOnly:          viper.GetBool("read-only"),
				ContentWindowSize: viper.GetInt("content-window-size"),
				ListenAddress:     viper.GetString("listen-address"),
				EndpointPath:      viper.GetString("http-path"),
				HealthPath:        viper.GetString("health-path"),
				ShutdownTimeout:   viper.GetDuration("shutdown-timeout"),
				LogFilePath:       viper.GetString("log-file"),
			}
			return ghmcp.RunHTTPServer(httpServerConfig)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetGlobalNormalizationFunc(wordSepNormalizeFunc)

	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Add global flags that will be shared by all commands
	rootCmd.PersistentFlags().StringSlice("toolsets", github.DefaultTools, "An optional comma separated list of groups of tools to allow, defaults to enabling all")
	rootCmd.PersistentFlags().Bool("dynamic-toolsets", false, "Enable dynamic toolsets")
	rootCmd.PersistentFlags().Bool("read-only", false, "Restrict the server to read-only operations")
	rootCmd.PersistentFlags().String("log-file", "", "Path to log file")
	rootCmd.PersistentFlags().Bool("enable-command-logging", false, "When enabled, the server will log all command requests and responses to the log file")
	rootCmd.PersistentFlags().Bool("export-translations", false, "Save translations to a JSON file")
	rootCmd.PersistentFlags().String("gh-host", "", "Specify the GitHub hostname (for GitHub Enterprise etc.)")
	rootCmd.PersistentFlags().Int("content-window-size", 5000, "Specify the content window size")

	// Bind flag to viper
	_ = viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))
	_ = viper.BindPFlag("dynamic_toolsets", rootCmd.PersistentFlags().Lookup("dynamic-toolsets"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("enable-command-logging", rootCmd.PersistentFlags().Lookup("enable-command-logging"))
	_ = viper.BindPFlag("export-translations", rootCmd.PersistentFlags().Lookup("export-translations"))
	_ = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("gh-host"))
	_ = viper.BindPFlag("content-window-size", rootCmd.PersistentFlags().Lookup("content-window-size"))

	httpCmd.Flags().String("listen", ":8080", "Address for the HTTP server to listen on")
	httpCmd.Flags().String("http-path", "/mcp", "HTTP path for MCP requests")
	httpCmd.Flags().String("health-path", "/health", "HTTP path for health checks")
	httpCmd.Flags().Duration("shutdown-timeout", 10*time.Second, "Graceful shutdown timeout for the HTTP server")

	_ = viper.BindPFlag("listen-address", httpCmd.Flags().Lookup("listen"))
	_ = viper.BindPFlag("http-path", httpCmd.Flags().Lookup("http-path"))
	_ = viper.BindPFlag("health-path", httpCmd.Flags().Lookup("health-path"))
	_ = viper.BindPFlag("shutdown-timeout", httpCmd.Flags().Lookup("shutdown-timeout"))

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(httpCmd)
}

func initConfig() {
	// Initialize Viper configuration
	viper.SetEnvPrefix("github")
	viper.AutomaticEnv()

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func wordSepNormalizeFunc(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	from := []string{"_"}
	to := "-"
	for _, sep := range from {
		name = strings.ReplaceAll(name, sep, to)
	}
	return pflag.NormalizedName(name)
}
