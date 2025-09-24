package utils

import (
"testing"
)

func TestInitLogger(t *testing.T) {
tests := []struct {
name   string
level  string
format string
wantErr bool
}{
{
name:   "JSON format with info level",
level:  "info",
format: "json",
wantErr: false,
},
{
name:   "Text format with debug level",
level:  "debug", 
format: "text",
wantErr: false,
},
{
name:   "Default level with unknown format",
level:  "unknown",
format: "console",
wantErr: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := InitLogger(tt.level, tt.format)
if (err != nil) != tt.wantErr {
t.Errorf("InitLogger() error = %v, wantErr %v", err, tt.wantErr)
}

if err == nil && Logger == nil {
t.Error("Logger should not be nil after successful initialization")
}
})
}
}

func TestLoggerFunctions(t *testing.T) {
// Initialize logger for testing
err := InitLogger("info", "json")
if err != nil {
t.Fatalf("Failed to initialize logger: %v", err)
}

// Test logging functions (should not panic)
t.Run("LogServerStart", func(t *testing.T) {
defer func() {
if r := recover(); r != nil {
t.Errorf("LogServerStart panicked: %v", r)
}
}()
LogServerStart(9000, "localhost")
})

t.Run("LogServerStop", func(t *testing.T) {
defer func() {
if r := recover(); r != nil {
t.Errorf("LogServerStop panicked: %v", r)
}
}()
LogServerStop()
})

t.Run("LogVideoStream", func(t *testing.T) {
defer func() {
if r := recover(); r != nil {
t.Errorf("LogVideoStream panicked: %v", r)
}
}()
LogVideoStream("test:video", "127.0.0.1", true)
LogVideoStream("test:video", "127.0.0.1", false)
})
}
