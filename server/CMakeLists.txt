cmake_minimum_required(VERSION 2.6)
project(peek)

macro(add_go_target executable_name directory_name)
    add_custom_target(${executable_name} ALL
        COMMAND rm -rf ${CMAKE_CURRENT_SOURCE_DIR}/go/pkg
        COMMAND GOPATH=${CMAKE_CURRENT_SOURCE_DIR}/go CGO_CFLAGS='-I${CMAKE_CURRENT_SOURCE_DIR}' CGO_LDFLAGS='-L${CMAKE_CURRENT_BINARY_DIR}' go get peek/${directory_name}
        COMMAND GOPATH=${CMAKE_CURRENT_SOURCE_DIR}/go CGO_CFLAGS='-I${CMAKE_CURRENT_SOURCE_DIR}' CGO_LDFLAGS='-L${CMAKE_CURRENT_BINARY_DIR}' go build peek/${directory_name}
    ${ARGN})
endmacro(add_go_target)

add_go_target(server httpserver)
