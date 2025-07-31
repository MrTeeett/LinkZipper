#!/usr/bin/env bash
set -e

BUILD_DIR="cmake-build"

case "$1" in
  	build)
		cmake -S . -B "$BUILD_DIR"
		cmake --build "$BUILD_DIR" --target build_all
	;;
  	clean)
		if [ -d "$BUILD_DIR" ]; then
	  		cmake --build "$BUILD_DIR" --target clean || true
		fi
		rm -rf "$BUILD_DIR" build CMakeFiles CMakeCache.txt Makefile cmake_install.cmake
		;;
  	*)
	echo "Usage: $0 {build|clean}"
	exit 1
	;;
esac