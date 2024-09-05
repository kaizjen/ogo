@echo off

go test github.com/kaizjen/ogo-disconnected/pkg/lib --count=1 -p 1
REM --count=1 disables caching
REM -p 1 needed so that tests aren't run in parallel