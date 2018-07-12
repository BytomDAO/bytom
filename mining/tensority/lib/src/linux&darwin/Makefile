TARGET = cSimdTs
CXX = g++
CXXFLAGS = -std=c++11 -pthread -mavx2 -O3 -fopenmp -fPIC

OS = linux
ARCH = amd64

.PHONY: clean

all: cSimdTs_$(OS)_$(ARCH)
	go build -buildmode=plugin -o simd_plugin_$(OS)_$(ARCH).so main.go

cSimdTs_$(OS)_$(ARCH): $(TARGET).cpp
	$(CXX) -o $@.o -c $^ $(CXXFLAGS)

clean:
	rm -f *.o *.so *.a