@echo off
set BUILD_DIR=cmake-build

if "%1"=="build_all" || "%2"=="build" (
	cmake -S . -B %BUILD_DIR%
	cmake --build %BUILD_DIR% --target build_all
) else if "%1"=="build_gui" (
	cmake -S . -B %BUILD_DIR%
	cmake --build %BUILD_DIR% --target build_gui
) else if "%1"=="build_api" (
	cmake -S . -B %BUILD_DIR%
	cmake --build %BUILD_DIR% --target build_api
) else if "%1"=="clean" (
	if exist %BUILD_DIR% (
		cmake --build %BUILD_DIR% --target clean >NUL 2>&1
		rmdir /s /q %BUILD_DIR%
	)
	if exist build rmdir /s /q build
) else (
	echo Usage: %0 build^|clean^|build_api^|build_gui
	exit /b 1
)