TARGET = cSimdTs

ifeq ($(OS),Windows_NT)
	$(error "Windows is not supported")
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		OS = linux
		CXX = g++
	endif
	ifeq ($(UNAME_S),Darwin)
		OS = darwin
		CXX = /usr/local/opt/llvm/bin/clang++
	endif
	UNAME_P := $(shell uname -p)
	ifeq ($(UNAME_P),x86_64)
		ARCH = amd64
	endif
	ifneq ($(filter %86,$(UNAME_P)),)
		ARCH = 386
	endif
	ifneq ($(filter arm%,$(UNAME_P)),)
		$(error "ARM is not supported")
	endif
endif

CXXFLAGS = -std=c++11 -pthread -mavx2 -O3 -fopenmp -fPIC

.PHONY: clean

all: $(TARGET)_$(OS)_$(ARCH)
	go build -buildmode=plugin -o simd_plugin_$(OS)_$(ARCH).so main.go

$(TARGET)_$(OS)_$(ARCH): $(TARGET).cpp
	$(CXX) -o $@.o -c $^ $(CXXFLAGS)

clean:
	rm -f *.o *.so *.a