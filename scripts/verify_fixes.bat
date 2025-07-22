@echo off
echo ========================================
echo Verifying Fleet Management System Fixes
echo ========================================
echo.

echo Running comprehensive diagnostic...
echo.

go run utilities/claude_doctor_v2.go

echo.
echo ========================================
echo Diagnostic complete!
echo.
echo If you see failures above:
echo 1. Ensure server is running (go run .)
echo 2. Check RESTART_REQUIRED.md for details
echo 3. Review server logs for errors
echo.
pause