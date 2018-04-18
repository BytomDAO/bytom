#ifndef BYTOMPOW_H
#define BYTOMPOW_H

#include "scrypt.h"
#include "sha3-allInOne.h"
#include <iostream>
#include <vector>
#include <time.h>
#include <assert.h>
#include <stdint.h>
#include <x86intrin.h>
#ifdef _USE_OPENMP
    #include <omp.h>
#endif


#define FNV(v1,v2) int32_t( ((v1)*FNV_PRIME) ^ (v2) )
const int FNV_PRIME = 0x01000193;

struct Mat256x256i8 {
    int8_t d[256][256];

    void toIdentityMatrix() {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                d[i][j]= (i==j)?1:0; // diagonal
            }
        }
    }

    void copyFrom(const Mat256x256i8& other) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                this->d[j][i]=other.d[j][i];
            }
        }
    }

    Mat256x256i8() {
//        this->toIdentityMatrix();
    }

    Mat256x256i8(const Mat256x256i8& other) {
        this->copyFrom(other);
    }

    void copyFrom_helper(LTCMemory& ltcMem, int offset) {
        for(int i=0; i<256; i++) {
            const Words32& lo=ltcMem.get(i*4+offset);
            const Words32& hi=ltcMem.get(i*4+2+offset);
            for(int j=0; j<64; j++) {
                uint32_t i32=j>=32?hi.get(j-32):lo.get(j);
                d[j*4+0][i]=(i32>>0)&0xFF;
                d[j*4+1][i]=(i32>>8)&0xFF;
                d[j*4+2][i]=(i32>>16)&0xFF;
                d[j*4+3][i]=(i32>>24)&0xFF;
            }
        }
    }

    void copyFromEven(LTCMemory& ltcMem) {
        copyFrom_helper(ltcMem, 0);
    }

    void copyFromOdd(LTCMemory& ltcMem) {
        copyFrom_helper(ltcMem, 1);
    }

    void add(Mat256x256i8& a, Mat256x256i8& b) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                int tmp=int(a.d[i][j])+int(b.d[i][j]);
                this->d[i][j]=(tmp&0xFF);
            }
        }
    }
};

struct Mat256x256i16 {
    int16_t d[256][256];

    void toIdentityMatrix() {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                d[i][j]=(i==j?1:0); // diagonal
            }
        }
    }

    void copyFrom(const Mat256x256i8& other) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                this->d[j][i]=int16_t(other.d[j][i]);
                assert(this->d[j][i]==other.d[j][i]);
            }
        }
    }

    void copyFrom(const Mat256x256i16& other) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                this->d[j][i]=other.d[j][i];
            }
        }
    }

    Mat256x256i16() {
//        this->toIdentityMatrix();
    }

    Mat256x256i16(const Mat256x256i16& other) {
        this->copyFrom(other);
    }

    void copyFrom_helper(LTCMemory& ltcMem, int offset) {
        for(int i=0; i<256; i++) {
            const Words32& lo=ltcMem.get(i*4+offset);
            const Words32& hi=ltcMem.get(i*4+2+offset);
            for(int j=0; j<64; j++) {
                uint32_t i32=j>=32?hi.get(j-32):lo.get(j);
                d[j*4+0][i]=int8_t((i32>>0)&0xFF);
                d[j*4+1][i]=int8_t((i32>>8)&0xFF);
                d[j*4+2][i]=int8_t((i32>>16)&0xFF);
                d[j*4+3][i]=int8_t((i32>>24)&0xFF);
            }
        }
    }

    void copyFromEven(LTCMemory& ltcMem) {
        copyFrom_helper(ltcMem, 0);
    }

    void copyFromOdd(LTCMemory& ltcMem) {
        copyFrom_helper(ltcMem, 1);
    }

    void mul(const Mat256x256i16& a, const Mat256x256i16& b) {
        for(int i=0; i<256; i+=16) {
            for(int j=0; j<256; j+=16) {
                for(int ii=i; ii<i+16; ii+=8) {
                    __m256i r[8],s,t[8],u[8],m[8];
                    r[0]=_mm256_set1_epi16(0);
                    r[1]=_mm256_set1_epi16(0);
                    r[2]=_mm256_set1_epi16(0);
                    r[3]=_mm256_set1_epi16(0);
                    r[4]=_mm256_set1_epi16(0);
                    r[5]=_mm256_set1_epi16(0);
                    r[6]=_mm256_set1_epi16(0);
                    r[7]=_mm256_set1_epi16(0);
                    for(int k=0; k<256; k++) {
                        s=*((__m256i*)(&(b.d[k][j])));
                        u[0]=_mm256_set1_epi16(a.d[ii+0][k]);
                        u[1]=_mm256_set1_epi16(a.d[ii+1][k]);
                        u[2]=_mm256_set1_epi16(a.d[ii+2][k]);
                        u[3]=_mm256_set1_epi16(a.d[ii+3][k]);
                        u[4]=_mm256_set1_epi16(a.d[ii+4][k]);
                        u[5]=_mm256_set1_epi16(a.d[ii+5][k]);
                        u[6]=_mm256_set1_epi16(a.d[ii+6][k]);
                        u[7]=_mm256_set1_epi16(a.d[ii+7][k]);
                        m[0]=_mm256_mullo_epi16(u[0],s);
                        m[1]=_mm256_mullo_epi16(u[1],s);
                        m[2]=_mm256_mullo_epi16(u[2],s);
                        m[3]=_mm256_mullo_epi16(u[3],s);
                        m[4]=_mm256_mullo_epi16(u[4],s);
                        m[5]=_mm256_mullo_epi16(u[5],s);
                        m[6]=_mm256_mullo_epi16(u[6],s);
                        m[7]=_mm256_mullo_epi16(u[7],s);
                        r[0]=_mm256_add_epi16(r[0],m[0]);
                        r[1]=_mm256_add_epi16(r[1],m[1]);
                        r[2]=_mm256_add_epi16(r[2],m[2]);
                        r[3]=_mm256_add_epi16(r[3],m[3]);
                        r[4]=_mm256_add_epi16(r[4],m[4]);
                        r[5]=_mm256_add_epi16(r[5],m[5]);
                        r[6]=_mm256_add_epi16(r[6],m[6]);
                        r[7]=_mm256_add_epi16(r[7],m[7]);
                    }
                    t[0]=_mm256_slli_epi16(r[0],8);
                    t[1]=_mm256_slli_epi16(r[1],8);
                    t[2]=_mm256_slli_epi16(r[2],8);
                    t[3]=_mm256_slli_epi16(r[3],8);
                    t[4]=_mm256_slli_epi16(r[4],8);
                    t[5]=_mm256_slli_epi16(r[5],8);
                    t[6]=_mm256_slli_epi16(r[6],8);
                    t[7]=_mm256_slli_epi16(r[7],8);
                    t[0]=_mm256_add_epi16(r[0],t[0]);
                    t[1]=_mm256_add_epi16(r[1],t[1]);
                    t[2]=_mm256_add_epi16(r[2],t[2]);
                    t[3]=_mm256_add_epi16(r[3],t[3]);
                    t[4]=_mm256_add_epi16(r[4],t[4]);
                    t[5]=_mm256_add_epi16(r[5],t[5]);
                    t[6]=_mm256_add_epi16(r[6],t[6]);
                    t[7]=_mm256_add_epi16(r[7],t[7]);
                    for(int x=0; x<8; x++) {
                        this->d[ii+x][j+0 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*0 +1)));
                        this->d[ii+x][j+1 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*1 +1)));
                        this->d[ii+x][j+2 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*2 +1)));
                        this->d[ii+x][j+3 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*3 +1)));
                        this->d[ii+x][j+4 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*4 +1)));
                        this->d[ii+x][j+5 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*5 +1)));
                        this->d[ii+x][j+6 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*6 +1)));
                        this->d[ii+x][j+7 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*7 +1)));
                        this->d[ii+x][j+8 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*8 +1)));
                        this->d[ii+x][j+9 ]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*9 +1)));
                        this->d[ii+x][j+10]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*10+1)));
                        this->d[ii+x][j+11]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*11+1)));
                        this->d[ii+x][j+12]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*12+1)));
                        this->d[ii+x][j+13]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*13+1)));
                        this->d[ii+x][j+14]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*14+1)));
                        this->d[ii+x][j+15]=int16_t(int8_t(_mm256_extract_epi8(t[x],2*15+1)));
                    }
                }
            }
        }
    }

    void add(Mat256x256i16& a, Mat256x256i16& b) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                int tmp=int(a.d[i][j])+int(b.d[i][j]);
                this->d[i][j]=(tmp&0xFF);
            }
        }
    }

    void toMatI8(Mat256x256i8& other) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                other.d[j][i]=(this->d[j][i])&0xFF;
            }
        }
    }

    void topup(Mat256x256i8& other) {
        for(int i=0; i<256; i++) {
            for(int j=0; j<256; j++) {
                other.d[j][i]+=(this->d[j][i])&0xFF;
            }
        }
    }
};


struct Arr256x64i32 {
    uint32_t d[256][64];

    uint8_t* d0RawPtr() {
        return (uint8_t*)(d[0]);
    }

    Arr256x64i32(const Mat256x256i8& mat) {
        for(int j=0; j<256; j++) {
            for(int i=0; i<64; i++) {
                d[j][i] = ((uint32_t(uint8_t(mat.d[j][i  + 192]))) << 24) |
                          ((uint32_t(uint8_t(mat.d[j][i + 128]))) << 16) |
                          ((uint32_t(uint8_t(mat.d[j][i  + 64]))) << 8) |
                          ((uint32_t(uint8_t(mat.d[j][i ]))) << 0);
            }
        }
    }

    void reduceFNV() {
        for(int k=256; k>1; k=k/2) {
            for(int j=0; j<k/2; j++) {
                for(int i=0; i<64; i++) {
                    d[j][i] = FNV(d[j][i], d[j + k / 2][i]);
                }
            }
        }
    }
};

// struct BytomMatList8 {
//     std::vector<Mat256x256i8*> matVec;

//     Mat256x256i8 at(int i) {
//         return *(matVec[i]);
//     }

//     BytomMatList8() {
//         for(int i=0; i<256; i++) {
//             Mat256x256i8* ptr = new Mat256x256i8;
//             assert(ptr!=NULL);
//             matVec.push_back(ptr);
//         }
//     }

//     ~BytomMatList8() {
//         for(int i=0; i<256; i++) {
//             delete matVec[i];
//         }
//     }

//     void init(const Words32& X_in) {
//         Words32 X = X_in;
//         LTCMemory ltcMem;
//         for(int i=0; i<128; i++) {
//             ltcMem.scrypt(X);
//             matVec[2*i]->copyFromEven(ltcMem);
//             matVec[2*i+1]->copyFromOdd(ltcMem);
//         }
//     }
// };

struct BytomMatList16 {
    std::vector<Mat256x256i16*> matVec;

    Mat256x256i16 at(int i) {
        return *(matVec[i]);
    }

    BytomMatList16() {
        for(int i=0; i<256; i++) {
            Mat256x256i16* ptr=new Mat256x256i16;
            assert(ptr!=NULL);
            matVec.push_back(ptr);
        }
    }

    ~BytomMatList16() {
        for(int i=0; i<256; i++)
            delete matVec[i];
    }

    void init(const Words32& X_in) {
        Words32 X = X_in;
        LTCMemory ltcMem;
        for(int i=0; i<128; i++) {
            ltcMem.scrypt(X);
            matVec[2*i]->copyFromEven(ltcMem);
            matVec[2*i+1]->copyFromOdd(ltcMem);
        }
    }

    // void copyFrom(BytomMatList8& other) {
    //     for(int i=0; i<256; i++) {
    //         matVec[i]->copyFrom(*other.matVec[i]);
    //     }
    // }

    void copyFrom(BytomMatList16& other) {
        for(int i=0; i<256; i++) {
            matVec[i]->copyFrom(*other.matVec[i]);
        }
    }
};

// extern BytomMatList8* matList_int8;
extern BytomMatList16* matList_int16;

static inline void iter_mineBytom(
                        const uint8_t *fixedMessage,
                        uint32_t len,
                        // uint8_t nonce[8],
                        uint8_t result[32]) {
    Mat256x256i8 *resArr8=new Mat256x256i8[4];

    clock_t start, end;
    start = clock();
    // Itz faster using single thread ...
#ifdef _USE_OPENMP
#pragma omp parallel for simd
#endif
    for(int k=0; k<4; k++) { // The k-loop
        sha3_ctx *ctx = new sha3_ctx;
        Mat256x256i16 *mat16=new Mat256x256i16;
        Mat256x256i16 *tmp16=new Mat256x256i16;
        uint8_t sequence[32];
        rhash_sha3_256_init(ctx);
        rhash_sha3_update(ctx, fixedMessage+(len*k/4), len/4);//分四轮消耗掉fixedMessage
        rhash_sha3_final(ctx, sequence);
        tmp16->toIdentityMatrix();

        for(int j=0; j<2; j++) {
            // equivalent as tmp=tmp*matlist, i+=1 
            for(int i=0; i<32; i+=2) {
                // "mc = ma dot mb.T" in GoLang code
                mat16->mul(*tmp16, matList_int16->at(sequence[i]));
                // "ma = mc" in GoLang code
                tmp16->mul(*mat16, matList_int16->at(sequence[i+1]));
            }
        }
        // "res[k] = mc" in GoLang code
        tmp16->toMatI8(resArr8[k]); // 0.00018s
        delete mat16;
        delete tmp16;
        delete ctx;
    }

    // 3.7e-05s
    Mat256x256i8 *res8=new Mat256x256i8;
    res8->add(resArr8[0], resArr8[1]);
    res8->add(*res8, resArr8[2]);
    res8->add(*res8, resArr8[3]);

    end = clock();    
    // std::cout << "\tTime for getting MulMatix: "
    //           << (double)(end - start) / CLOCKS_PER_SEC * 1000 << "ms"
    //           << std::endl;

    Arr256x64i32 arr(*res8);
    arr.reduceFNV();
    sha3_ctx *ctx = new sha3_ctx;
    rhash_sha3_256_init(ctx);
    rhash_sha3_update(ctx, arr.d0RawPtr(), 256);
    rhash_sha3_final(ctx, result);

    delete res8;
    delete[] resArr8;
    delete ctx;
}

static inline void incrNonce(uint8_t nonce[8]) {
    for(int i=0; i<8; i++) {
        if(nonce[i]!=255) {
            nonce[i]++;
            break;
        } else {
            nonce[i]=0;
        }
    }
}

static inline int countLeadingZero(uint8_t result[32]) {
    int count=0;
    for (int i=31; i>=0; i--) { // NOTE: reverse
        if (result[i] < 1) {
            count+=8;
        } else if (result[i]<2)  {
            count+=7;
            break;
        } else if (result[i]<4)  {
            count+=6;
            break;
        } else if (result[i]<8)  {
            count+=5;
            break;
        } else if (result[i]<16) {
            count+=4;
            break;
        } else if (result[i]<32) {
            count+=3;
            break;
        } else if (result[i]<64) {
            count+=2;
            break;
        } else if (result[i]<128) {
            count+=1;
            break;
        }
    }
    return count;
}

// static inline int test_mineBytom(
//     const uint8_t *fixedMessage,
//     uint32_t len,
//     uint8_t nonce[32],
//     int count,
//     int leadingZeroThres)
// {
//   assert(len%4==0);
//   int step;
//   for(step=0; step<count; step++) {
//     uint8_t result[32];
//     //std::cerr<<"Mine step "<<step<<std::endl;
//     iter_mineBytom(fixedMessage,100,nonce,result);
//     std::cerr<<"Mine step "<<step<<std::endl;
//     for (int i = 0; i < 32; i++) {
//       printf("%02x ", result[i]);
//       if (i % 8 == 7)
//         printf("\n");
//     }
//     if (countLeadingZero(result) > leadingZeroThres)
//       return step;
//     incrNonce(nonce);
//   }
//   return step;
// }


#endif

