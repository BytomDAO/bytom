TARGET = simd

ifeq ($(OS),Windows_NT)
    CXX = g++
    ifeq ($(PROCESSOR_ARCHITEW6432),AMD64)
        ARCH = amd64
    else
        ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
            ARCH = amd64
        endif
        ifeq ($(PROCESSOR_ARCHITECTURE),x86)
            ARCH = 386
        endif
    endif
else
    $(error "Other OS than win is not supported")
endif

CXXFLAGS = -std=c++11 -pthread -mavx2 -O3 -fopenmp -fPIC

.PHONY: clean

all: $(TARGET)_plugin_win_$(ARCH)

$(TARGET)_win_$(ARCH): dllEntryPoint.o c$(TARGET)Ts.o
	$(CXX) -o $@.dll $^

*.o: *.cpp
	$(CXX) -o $@.o -c $^ $(CXXFLAGS)

test: all
	cp $(TARGET)_win_$(ARCH).dll  ../../../
	cd ../../../ && go test -v && go test -bench=. && rm -f $(TARGET)_win_$(ARCH).dll

clean:
	rm -f *.o *.dll *.a *.so