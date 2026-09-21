/*
     Copyright 2024 Felix Weinmann

     Licensed under the Apache License, Version 2.0 (the "License");
     you may not use this file except in compliance with the License.
     You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

     Unless required by applicable law or agreed to in writing, software
     distributed under the License is distributed on an "AS IS" BASIS,
     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
     See the License for the specific language governing permissions and
     limitations under the License.
 */
#ifndef CL_VERSION_1_0
#include "clstd.h"
#endif

const sampler_t sampler = CLK_FILTER_NEAREST | CLK_NORMALIZED_COORDS_FALSE | CLK_ADDRESS_CLAMP_TO_EDGE;

inline float read(read_only image2d_t sat, int2 pos, const int dx, const int dy) {
	pos.x += dx;
	pos.y += dy;

	return read_imagef(sat, sampler, pos).x;
}

//https://en.wikipedia.org/wiki/Summed-area_table
//https://en.wikipedia.org/wiki/Prefix_sum#Applications
//https://dl.acm.org/doi/abs/10.5555/2346696.2346743
//https://blog.demofox.org/2018/04/16/prefix-sums-and-summed-area-tables/
//https://github.com/Algomorph/clsat https://github.com/Algomorph/clsat/blob/master/src/sat.cl
kernel void circle(read_only image2d_t sat, write_only image2d_t out, int maxBlobRadius) {
	int2 pos = (int2)(get_global_id(0), get_global_id(1));

	float ppScore = read(sat, pos,  maxBlobRadius,  maxBlobRadius) - read(sat, pos,  maxBlobRadius,  1) - read(sat, pos,  1,  maxBlobRadius) + read(sat, pos,  1,  1);
	float pnScore = read(sat, pos,  maxBlobRadius, -maxBlobRadius) - read(sat, pos,  maxBlobRadius, -1) - read(sat, pos,  1, -maxBlobRadius) + read(sat, pos,  1, -1); //inverted
	float npScore = read(sat, pos, -maxBlobRadius,  maxBlobRadius) - read(sat, pos, -maxBlobRadius,  1) - read(sat, pos, -1,  maxBlobRadius) + read(sat, pos, -1,  1); //inverted
	float nnScore = read(sat, pos, -maxBlobRadius, -maxBlobRadius) - read(sat, pos, -maxBlobRadius, -1) - read(sat, pos, -1, -maxBlobRadius) + read(sat, pos, -1, -1);
	write_imagef(out, pos, min(min(ppScore, nnScore), min(pnScore, npScore)) / (maxBlobRadius*maxBlobRadius));
}
