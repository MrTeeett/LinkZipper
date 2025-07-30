cmake_minimum_required(VERSION 3.15)
project(linkzipper NONE)

# Определяем суффикс для .exe на Windows
if (CMAKE_SYSTEM_NAME STREQUAL "Windows")
  set(EXE_SUFFIX ".exe")
else()
  set(EXE_SUFFIX "")
endif()

# Основная выходная директория: build/<Система>/
set(OUTPUT_BASE "${CMAKE_SOURCE_DIR}/build/${CMAKE_SYSTEM_NAME}")

# Создаём директории автоматически при конфигурации
file(MAKE_DIRECTORY "${OUTPUT_BASE}/api")
file(MAKE_DIRECTORY "${OUTPUT_BASE}/gui")

# Таргет для сборки API
add_custom_target(build_all
  COMMAND ${CMAKE_COMMAND} -E echo "Building API server for ${CMAKE_SYSTEM_NAME}..."
  COMMAND go build
          -o "${OUTPUT_BASE}/api/archive_service_api${EXE_SUFFIX}"
          "${CMAKE_SOURCE_DIR}/cmd/api/main.go"
  WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
  COMMENT ">>> build_api"
  VERBATIM
)

