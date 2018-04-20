#ifndef SEED_H
#define SEED_H

static inline void extend(uint32_t* exted, uint8_t *g_seed){
    sha3_ctx *ctx = (sha3_ctx*)calloc(1, sizeof(*ctx));
    // uint8_t seedHash[4*32];
    uint8_t seedHash[4][32];

    //  std::copy beats memcpy
    // std::copy(g_seed, g_seed + 32, seedHash);
    std::copy(g_seed, g_seed + 32, seedHash[0]);
    
    for(int i = 0; i < 3; ++i) {
        rhash_sha3_256_init(ctx);
        // rhash_sha3_update(ctx, seedHash+i*32, 32);
        // rhash_sha3_final(ctx, seedHash+(i+1)*32);
        rhash_sha3_update(ctx, seedHash[i], 32);
        rhash_sha3_final(ctx, seedHash[i+1]);
    }

    for(int i = 0; i < 32; ++i) {
//        exted[i] =  ((*(seedHash+i*4+3))<<24) +
//                    ((*(seedHash+i*4+2))<<16) +
//                    ((*(seedHash+i*4+1))<<8) +
//                    (*(seedHash+i*4));
        exted[i] =  (seedHash[i/8][(i*4+3)%32]<<24) +
                    (seedHash[i/8][(i*4+2)%32]<<16) +
                    (seedHash[i/8][(i*4+1)%32]<<8) +
                    seedHash[i/8][(i*4)%32];
    }

    free(ctx);
}

static inline void init_seed(Words32 &seed, uint32_t _seed[32])
{
    for (int i = 0; i < 16; i++)
        seed.lo.w[i] = _seed[i];
    for (int i = 0; i < 16; i++)
        seed.hi.w[i] = _seed[16 + i];
}

#endif