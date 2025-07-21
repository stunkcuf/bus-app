@echo off
echo Checking Fleet Management System Port Configuration
echo ==================================================
echo.

if defined PORT (
    echo PORT environment variable is set to: %PORT%
    echo The application will run on: http://localhost:%PORT%
) else (
    echo PORT environment variable is NOT set
    echo The application will use default port: 5000
    echo The application will run on: http://localhost:5000
)

echo.
echo Testing connection to common ports...
echo.

curl -s -o nul -w "Port 5000: %%{http_code}\n" http://localhost:5000/health 2>nul || echo Port 5000: Not responding
curl -s -o nul -w "Port 5003: %%{http_code}\n" http://localhost:5003/health 2>nul || echo Port 5003: Not responding

echo.
echo If you want to use port 5003, run:
echo   set PORT=5003
echo   go run .
echo.
pause