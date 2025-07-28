@echo off
echo Building hs-bus...
go build -o hs-bus.exe 2> build_errors.txt
if %ERRORLEVEL% NEQ 0 (
    echo Build failed. Errors saved to build_errors.txt
    type build_errors.txt
) else (
    echo Build successful!
)