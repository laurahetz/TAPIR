#include "regular_dpf.h"
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// int foo () {
//     // Test the keyGen function
//     u64 domain = 1;
//     u64 points[] = {1};
//     char* values[] = {"111111111111111"};
//     size_t numPoints = 1;
//     char* prngSeed =  {"111111111111111"}; // 16 bytes for PRNG seed
//     u64 keySize;
    
//     unsigned char* keys = keyGen(domain, points, values, numPoints, prngSeed, &keySize);
    
//     printf("keyGen result: ");
//     for (size_t i = 0; i < keySize; ++i) {
//         printf("%02X ", keys[i]);
//     }
//     printf("\n");

//     // Free allocated memory for keys
//     free(keys);

//     return 0;
// }