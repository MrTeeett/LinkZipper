#!/usr/bin/env bash
set -e

BUILD_DIR="cmake-build"

case "$1" in
	build)
		cmake -S . -B "$BUILD_DIR"
		cmake --build "$BUILD_DIR" --target build_all
	;;
  	build_all)
		cmake -S . -B "$BUILD_DIR"
		cmake --build "$BUILD_DIR" --target build_all
	;;
	build_gui)
		cmake -S . -B "$BUILD_DIR"
		cmake --build "$BUILD_DIR" --target build_gui
	;;
	build_api)
		cmake -S . -B "$BUILD_DIR"
		cmake --build "$BUILD_DIR" --target build_api
	;;
  	clean)
		if [ -d "$BUILD_DIR" ]; then
	  		cmake --build "$BUILD_DIR" --target clean || true
		fi
		rm -rf "$BUILD_DIR" build
		;;
  	*)
	echo "Usage: $0 {build|clean|build_api|build_gui}"
	exit 1
	;;
esac