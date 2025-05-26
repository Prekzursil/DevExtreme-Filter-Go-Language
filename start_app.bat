@echo off
echo Starting Go Backend Server...
echo Make sure you are in the 'transaction-filter-backend' directory.
echo If GCC is required for CGO, ensure it's in your PATH.

REM Set CGO_ENABLED=1 for SQLite
set CGO_ENABLED=1

echo Running: go run .
go run .

echo.
echo If the server started successfully, you can access:
echo   - Main Application (React UI): http://localhost:8080/
echo   - Developer Schema Editor: http://localhost:8080/schema-editor
echo.
pause
