package logger

import (
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func init(){
	lumberjackLogger := utils.NewLumberjackLogger("lazyspotify.log")
	Log = zerolog.New(lumberjackLogger).With().Timestamp().Caller().Logger()
	Log.Debug().Msg("logger initialized")
	Log.Debug().Msgf("config directory: %s", utils.SafeGetConfigDir())

}
