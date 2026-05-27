@echo off
chcp 65001 >nul 2>&1
title OIDC Platform

cd /d "%~dp0"

if not exist "configs\config.yaml" (
    echo [ERROR] configs\config.yaml not found
    pause
    exit /b 1
)

if not exist "data" mkdir data

if not exist "frontend\dist" (
    echo [WARN] frontend\dist not found, SPA will be disabled
)

echo ============================================
echo   OIDC Platform - SQLite Mode
echo   http://localhost:8080
echo   Admin: see configs\config.yaml
echo ============================================
echo.

set OIDC_OAUTH2_SECRET=3ade3492fcfb11c22340b4a53b10bfa728b2a5c724e5ac007247307ff440ee38

oidc.exe

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Server exited with code %errorlevel%
    pause
)
