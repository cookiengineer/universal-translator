cmake_minimum_required(VERSION 3.12.4)

project(bergamot_bridge CXX C)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

# --- Options for the vendored Marian / Bergamot / SentencePiece build ---------
# These must be in the cache before add_subdirectory() so the nested projects do
# not re-default them. We build a CPU-only, intgemm, single-threaded-friendly
# decoder-only library with no training pipeline.
set(COMPILE_CUDA OFF CACHE BOOL "Compile GPU version" FORCE)
set(COMPILE_CPU ON CACHE BOOL "Compile CPU version" FORCE)
set(USE_SENTENCEPIECE ON CACHE BOOL "Compile SentencePiece" FORCE)
# Kept OFF so Marian can discover the system OpenBLAS/CBLAS shared libraries for
# its float GEMMs (intgemm only covers the quantised int8 GEMMs). SentencePiece
# is built statically regardless (SPM_ENABLE_SHARED is forced off).
set(USE_STATIC_LIBS OFF CACHE BOOL "Link statically against non-system libs" FORCE)
set(USE_WASM_COMPATIBLE_SOURCE OFF CACHE BOOL "Do not use wasm-compatible sources" FORCE)
set(COMPILE_LIBRARY_ONLY ON CACHE BOOL "Build only the Marian library" FORCE)
set(USE_MKL OFF CACHE BOOL "Use MKL" FORCE)
set(USE_FBGEMM OFF CACHE BOOL "Use FBGEMM" FORCE)
set(USE_RUY OFF CACHE BOOL "Use Ruy" FORCE)
set(USE_ONNX_SGEMM OFF CACHE BOOL "Use ONNX SGEMM" FORCE)
set(USE_MPI OFF CACHE BOOL "Use MPI" FORCE)
set(USE_NCCL OFF CACHE BOOL "Use NCCL" FORCE)
set(USE_DOXYGEN OFF CACHE BOOL "Build docs" FORCE)
set(COMPILE_SERVER OFF CACHE BOOL "Compile marian-server" FORCE)
set(COMPILE_TESTS OFF CACHE BOOL "Compile tests" FORCE)
set(COMPILE_EXAMPLES OFF CACHE BOOL "Compile examples" FORCE)
set(COMPILE_UNIT_TESTS OFF CACHE BOOL "Compile unit tests" FORCE)
set(ENABLE_CACHE_STATS OFF CACHE BOOL "Enable cache stats" FORCE)
set(SSPLIT_COMPILE_LIBRARY_ONLY ON CACHE BOOL "Compile ssplit library only" FORCE)
set(SSPLIT_USE_INTERNAL_PCRE2 OFF CACHE BOOL "Use system PCRE2" FORCE)
# A portable x86-64-v3 (AVX2) baseline keeps the build machine-independent and
# avoids compiler bugs triggered by AVX-512 codegen. intgemm still performs its
# own runtime CPU dispatch for the int8 GEMM kernels.
set(BUILD_ARCH x86-64-v3 CACHE STRING "Compile for this CPU architecture" FORCE)
# Use OpenBLAS for float GEMMs instead of any MKL install that might also be
# present, keeping the runtime dependency small and predictable.
set(BLA_VENDOR OpenBLAS CACHE STRING "BLAS implementation to use" FORCE)
# Do not link tcmalloc; it is an optional allocator replacement that would
# interpose malloc inside the Go process.
set(CMAKE_DISABLE_FIND_PACKAGE_Tcmalloc TRUE CACHE BOOL "Disable tcmalloc" FORCE)
set(SPM_ENABLE_TCMALLOC OFF CACHE BOOL "Disable tcmalloc in sentencepiece" FORCE)

add_subdirectory(third_party/bergamot-translator)

# --- Our thin C ABI bridge ----------------------------------------------------
add_library(bergamot_bridge SHARED bridge/bridge.cpp)

target_include_directories(bergamot_bridge PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/bridge)

target_link_libraries(bergamot_bridge PRIVATE bergamot-translator)

# Keep the C++ runtime self-contained inside the shared object.
target_link_options(bergamot_bridge PRIVATE -static-libstdc++ -static-libgcc)

set_target_properties(bergamot_bridge PROPERTIES
  LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}"
  OUTPUT_NAME "bergamot_bridge")
