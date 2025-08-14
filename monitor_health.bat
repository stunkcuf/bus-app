@echo off
echo ========================================
echo Fleet Management System Health Monitor
echo ========================================
echo.

:loop
echo [%date% %time%] Checking system health...
echo.

echo 1. Server Status:
curl -s http://localhost:8080/health 2>nul || echo Server not responding
echo.
echo.

echo 2. Database Status:
curl -s http://localhost:8080/api/debug/data 2>nul | findstr /C:"error" /C:"Error" || echo Database OK
echo.

echo 3. GPS Status:
curl -s http://localhost:8080/api/gps/status 2>nul || echo GPS endpoint not responding
echo.

echo ----------------------------------------
timeout /t 10 >nul
goto loop