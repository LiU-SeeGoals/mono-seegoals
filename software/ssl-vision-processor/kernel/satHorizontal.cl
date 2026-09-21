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

const sampler_t sampler = CLK_FILTER_NEAREST | CLK_NORMALIZED_COORDS_FALSE | CLK_ADDRESS_NONE;

kernel void sat_horizontal(read_only image2d_t in, write_only image2d_t out) {
	const int width = get_image_width(in);
	const int y = get_global_id(0);

	float sum = 0.f;
	for(int x = 0; x < width; x++) {
		const int2 pos = (int2)(x, y);
		sum += read_imagef(in, sampler, pos).x;
		write_imagef(out, pos, sum);
	}
}