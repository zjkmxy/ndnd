/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/named-data/ndnd/std/log"
)

var Log = log.Default()
var logFileObj *os.File

// OpenLogger initializes the logger.
func OpenLogger(filename string) {
	// open file if filename is not empty
	if filename == "" {
		logFileObj = os.Stderr
	} else {
		var err error
		logFileObj, err = os.Create(filename)
		if err != nil {
			panic(err)
		}
	}

	// create new logger
	Log = log.NewText(logFileObj)

	// set log level
	level, err := log.ParseLevel(GetConfig().Core.LogLevel)
	if err != nil {
		panic(err)
	}
	Log.SetLevel(level)
}

// ShutdownLogger shuts down the logger.
func CloseLogger() {
	if logFileObj != nil {
		logFileObj.Close()
	}
}

func generateLogMessage(module interface{}, components ...interface{}) string {
	var message strings.Builder
	message.WriteString(fmt.Sprintf("[%v] ", module))
	for _, component := range components {
		switch v := component.(type) {
		case string:
			message.WriteString(v)
		case int:
			message.WriteString(strconv.Itoa(v))
		case int8:
			message.WriteString(strconv.FormatInt(int64(v), 10))
		case int16:
			message.WriteString(strconv.FormatInt(int64(v), 10))
		case int32:
			message.WriteString(strconv.FormatInt(int64(v), 10))
		case int64:
			message.WriteString(strconv.FormatInt(v, 10))
		case uint:
			message.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint8:
			message.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint16:
			message.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint32:
			message.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint64:
			message.WriteString(strconv.FormatUint(v, 10))
		case uintptr:
			message.WriteString(strconv.FormatUint(uint64(v), 10))
		case bool:
			message.WriteString(strconv.FormatBool(v))
		case error:
			message.WriteString(v.Error())
		default:
			message.WriteString(fmt.Sprintf("%v", component))
		}
	}
	return message.String()
}

// LogFatal logs a message at the FATAL level. Note: Fatal will let the program exit
func LogFatal(module interface{}, components ...interface{}) {
	log.Fatal(generateLogMessage(module, components...))
}

// LogError logs a message at the ERROR level.
func LogError(module interface{}, components ...interface{}) {
	log.Error(generateLogMessage(module, components...))
}

// LogWarn logs a message at the WARN level.
func LogWarn(module interface{}, components ...interface{}) {
	log.Warn(generateLogMessage(module, components...))
}

// LogDebug logs a message at the DEBUG level.
func LogDebug(module interface{}, components ...interface{}) {
	log.Debug(generateLogMessage(module, components...))
}

// LogTrace logs a message at the TRACE level (really just additional DEBUG messages).
func LogTrace(module interface{}, components ...interface{}) {
	log.Trace(generateLogMessage(module, components...))
}
