cmake_minimum_required(VERSION 3.15)
project(LinkZipper NONE)

if (CMAKE_SYSTEM_NAME STREQUAL "Windows")
	set(EXE_SUFFIX ".exe")
else()
	set(EXE_SUFFIX "")
endif()

set(OUTPUT_BASE "${CMAKE_SOURCE_DIR}/build/${CMAKE_SYSTEM_NAME}")

file(MAKE_DIRECTORY "${OUTPUT_BASE}/api")
file(MAKE_DIRECTORY "${OUTPUT_BASE}/gui")

add_custom_target(build_api
	COMMAND ${CMAKE_COMMAND} -E echo "Building API server for ${CMAKE_SYSTEM_NAME}..."
	COMMAND go build
			-o "${OUTPUT_BASE}/api/Link Zipper${EXE_SUFFIX}"
			"${CMAKE_SOURCE_DIR}/cmd/api/main.go"
	WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
	COMMENT ">>> build_api"
	VERBATIM
)

add_custom_target(build_gui
	COMMAND ${CMAKE_COMMAND} -E echo "Building GUI for ${CMAKE_SYSTEM_NAME}..."
	COMMAND go build
			-o "${OUTPUT_BASE}/gui/Link Zipper Gui${EXE_SUFFIX}"
			"${CMAKE_SOURCE_DIR}/cmd/gui/main.go"
	WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
	COMMENT ">>> build_gui"
	VERBATIM
)

add_custom_target(build_all
	DEPENDS build_api build_gui
	COMMENT ">>> build_all: сборка API и GUI"
)
