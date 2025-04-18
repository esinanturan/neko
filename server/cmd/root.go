package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	neko "github.com/m1k1o/neko/server"
	"github.com/m1k1o/neko/server/internal/config"
)

func Execute() error {
	// properly log unhandled panics
	defer func() {
		panicVal := recover()
		if panicVal != nil {
			log.Panic().Msgf("%v", panicVal)
		}
	}()

	return root.Execute()
}

var root = &cobra.Command{
	Use:     "neko",
	Short:   "neko streaming server",
	Long:    `neko streaming server`,
	Version: neko.Version.String(),
}

func init() {
	rootConfig := config.Root{}

	cobra.OnInitialize(func() {
		//////
		// configs
		//////

		config := viper.GetString("config") // Use config file from the flag.
		if config == "" {
			config = os.Getenv("NEKO_CONFIG") // Use config file from the environment variable.
		}

		if config != "" {
			viper.SetConfigFile(config)
		} else {
			if runtime.GOOS == "linux" {
				viper.AddConfigPath("/etc/neko/")
			}

			viper.AddConfigPath(".")
			viper.SetConfigName("neko")
		}

		viper.SetEnvPrefix("NEKO")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv() // read in environment variables that match

		// read config values
		err := viper.ReadInConfig()
		if err != nil {
			_, notFound := err.(viper.ConfigFileNotFoundError)
			if !notFound {
				log.Fatal().Err(err).Msg("unable to read config file")
			}
		}

		// get full config file path
		config = viper.ConfigFileUsed()

		// set root config values
		rootConfig.Set()

		// legacy if explicitly enabled or if unspecified and legacy config is found
		if viper.GetBool("legacy") || !viper.IsSet("legacy") {
			rootConfig.SetV2()
		}

		//////
		// logs
		//////
		var logWriter io.Writer

		// log to a directory instead of stderr
		if rootConfig.LogDir != "" {
			if _, err := os.Stat(rootConfig.LogDir); os.IsNotExist(err) {
				_ = os.Mkdir(rootConfig.LogDir, os.ModePerm)
			}

			latest := filepath.Join(rootConfig.LogDir, "neko-latest.log")
			if _, err := os.Stat(latest); err == nil {
				err = os.Rename(latest, filepath.Join(rootConfig.LogDir, "neko."+time.Now().Format("2006-01-02T15-04-05Z07-00")+".log"))
				if err != nil {
					log.Fatal().Err(err).Msg("failed to rotate log file")
				}
			}

			logf, err := os.OpenFile(latest, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to open log file")
			}

			logWriter = diode.NewWriter(logf, 1000, 10*time.Millisecond, func(missed int) {
				fmt.Printf("logger dropped %d messages", missed)
			})
		} else {
			logWriter = os.Stderr
		}

		// log console output instead of json
		if !rootConfig.LogJson {
			logWriter = zerolog.ConsoleWriter{
				Out:     logWriter,
				NoColor: rootConfig.LogNocolor,
			}
		}

		// save new logger output
		log.Logger = log.Output(logWriter)

		// set custom log level
		if rootConfig.LogLevel != zerolog.NoLevel {
			zerolog.SetGlobalLevel(rootConfig.LogLevel)
		}

		// set custom log tiem format
		if rootConfig.LogTime != "" {
			zerolog.TimeFieldFormat = rootConfig.LogTime
		}

		timeFormat := rootConfig.LogTime
		if rootConfig.LogTime == zerolog.TimeFormatUnix {
			timeFormat = "UNIX"
		}

		logger := log.With().
			Str("config", config).
			Str("log-level", zerolog.GlobalLevel().String()).
			Bool("log-json", rootConfig.LogJson).
			Str("log-time", timeFormat).
			Str("log-dir", rootConfig.LogDir).
			Logger()

		if config == "" {
			logger.Warn().Msg("preflight complete without config file")
		} else {
			if _, err := os.Stat(config); os.IsNotExist(err) {
				logger.Err(err).Msg("preflight complete with nonexistent config file")
			} else {
				logger.Info().Msg("preflight complete with config file")
			}
		}
	})

	if err := rootConfig.Init(root); err != nil {
		log.Panic().Err(err).Msg("unable to run root command")
	}

	// legacy if explicitly enabled or if unspecified and legacy config is found
	if viper.GetBool("legacy") || !viper.IsSet("legacy") {
		if err := rootConfig.InitV2(root); err != nil {
			log.Panic().Err(err).Msg("unable to run root command")
		}
	}

	root.SetVersionTemplate(neko.Version.Details())
}
