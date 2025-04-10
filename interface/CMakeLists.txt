# Define the directories
set(UI_SRC_DIR "${CMAKE_CURRENT_SOURCE_DIR}" CACHE INTERNAL "")
set(PUBLIC_DIR "${UI_SRC_DIR}/public")

# Get all UI files recursively
file(GLOB_RECURSE ALL_UI_FILES "${UI_SRC_DIR}/src/*")

# Get all test UI files (those matching the pattern '*.spec.*')
file(GLOB_RECURSE TEST_UI_FILES "${UI_SRC_DIR}/src/*.spec.*")

# Get the list of APP_UI_FILES by removing TEST_UI_FILES from ALL_UI_FILES
list(FILTER ALL_UI_FILES EXCLUDE REGEX ".*\\.spec\\..*")
set(APP_UI_FILES ${ALL_UI_FILES})

# Get all files in the PUBLIC_DIR (non-recursive)
file(GLOB PUBLIC_FILES "${PUBLIC_DIR}/*")

# Get the list of UI_CONFIG_FILES
file(GLOB UI_CONFIG_FILES
  "${UI_SRC_DIR}/*.json"
  "${UI_SRC_DIR}/*.config.*"
  "${UI_SRC_DIR}/pluginPWA.ts"
)

set(UI_DIST_DIR ${CMAKE_SOURCE_DIR}/server/ui/static)

set(UI_DIST
  ${UI_DIST_DIR}/assets
  ${UI_DIST_DIR}/index.html
  ${UI_DIST_DIR}/favicon.ico
  ${UI_DIST_DIR}/manifest.json
  ${UI_DIST_DIR}/robots.txt
  ${UI_DIST_DIR}/apple-touch-icon.png
  ${UI_DIST_DIR}/apple-touch-icon-precomposed.png
  CACHE INTERNAL ""
)
add_custom_command(
  OUTPUT ${UI_DIST}
  BYPRODUCTS ${UI_DIST}
  # Since output files are hashed, make sure that there aren't any files that might be left over from the previous build
  # in the output directory.
  COMMAND ${GIT_EXECUTABLE} clean -f -X ${UI_DIST_DIR}
  # Once the output directory is clean, then actually build the user interface.
  COMMAND ${RSBUILD_EXECUTABLE} build --mode production
  COMMAND ${CMAKE_COMMAND} -E make_directory ${UI_DIST_DIR}/assets/resources
  COMMENT "Building monetr's user interface"
  WORKING_DIRECTORY ${UI_SRC_DIR}
  VERBATIM
  DEPENDS
    dependencies.node_modules
    tools.rsbuild
    ${APP_UI_FILES}
    ${PUBLIC_FILES}
    ${UI_CONFIG_FILES}
)

add_custom_target(
  build.interface ALL
  DEPENDS ${UI_DIST}
)

add_custom_target(
  development.lite
  COMMAND ${CMAKE_COMMAND} -E env MONETR_DEVELOPMENT_LITE=true MONETR_DEVELOPMENT_LITE_TARGET=${MONETR_DEVELOPMENT_LITE_TARGET} ${RSBUILD_EXECUTABLE} dev
  COMMENT "Starting frontend lite local development"
  WORKING_DIRECTORY ${UI_SRC_DIR}
  DEPENDS
    dependencies.node_modules
    tools.rsbuild
    ${APP_UI_FILES}
    ${PUBLIC_FILES}
    ${UI_CONFIG_FILES}
)
