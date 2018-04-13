#include <iostream>
#include <cstdio>
#include <map>
#include "cSimdTs.h"
#include "BytomPoW.h"
#include "seed.h"

using namespace std;

BytomMatList16* matList_int16;
uint8_t result[32] = {0};
map <vector<uint8_t>, BytomMatList16*> seedCache;

uint8_t *SimdTs(uint8_t blockheader[32], uint8_t seed[32]){
    vector<uint8_t> seedVec(seed, seed + 32);

    if(seedCache.find(seedVec) != seedCache.end()) {
        // printf("\t---%s---\n", "Seed already exists in the cache.");
        matList_int16 = seedCache[seedVec];
    } else {
        uint32_t exted[32];
        extend(exted, seed); // extends seed to exted
        Words32 extSeed;
        init_seed(extSeed, exted);

        matList_int16 = new BytomMatList16;
        matList_int16->init(extSeed);

        seedCache.insert(pair<vector<uint8_t>, BytomMatList16*>(seedVec, matList_int16));
    }

    iter_mineBytom(blockheader, 32, result);
    
    return result;
}
