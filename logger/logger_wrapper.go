/**
 * Copyright (c) 2016 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package logger_wrapper

import (
	"github.com/op/go-logging"
	"os"
)

var logLevelStr = os.Getenv("BROKER_LOG_LEVEL")

func InitLogger(module string) *logging.Logger {

	var logLevel = logging.INFO
	if logLevelStr != "" {
		logLevelConverted, err := logging.LogLevel(logLevelStr)
		if err == nil {
			logLevel = logLevelConverted
		}

	}

	logger := logging.MustGetLogger(module)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.4s} ▶ [%{shortfunc}]: %{color:reset} %{message}`,
	)

	backend1 := logging.NewLogBackend(os.Stderr, "", 0)

	// For messages written to backend1 we want to add some additional
	// information to the output, including the used log level and the name of
	// the function.
	backend1Formatter := logging.NewBackendFormatter(backend1, format)

	// Only errors and more severe messages should be sent to backend1
	backend1Leveled := logging.AddModuleLevel(backend1Formatter)
	backend1Leveled.SetLevel(logLevel, module)

	// Set the backends to be used.
	logging.SetBackend(backend1Leveled)

	return logger
}
