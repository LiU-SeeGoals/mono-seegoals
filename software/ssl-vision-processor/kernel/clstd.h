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
#pragma once

#define kernel
#define global
#define local
#define constant
#define read_write
#define read_only
#define write_only

#define CLK_NORMALIZED_COORDS_FALSE 0
#define CLK_ADDRESS_NONE 1
#define CLK_ADDRESS_CLAMP_TO_EDGE 2
#define CLK_ADDRESS_CLAMP 3
#define CLK_FILTER_LINEAR 4
#define CLK_FILTER_NEAREST 5

typedef unsigned char uchar;
typedef struct { uchar x, y, z; } uchar3;
typedef unsigned int uint;
typedef struct { int x, y; } int2;
typedef struct { float x, y; } float2;
typedef struct { float x, y, z; } float3;
typedef struct { float x, y, z, w; } float4;
typedef struct { int x, y, z, w; } int4;
typedef struct { uint x, y, z, w; uint r, g, b; } uint4;
typedef int sampler_t;
typedef struct {} image2d_t;

uint4 read_imageui(image2d_t, sampler_t, int2);
uint4 read_imageui(image2d_t, sampler_t, float2);
int4 read_imagei(image2d_t, sampler_t, int2);
float4 read_imagef(image2d_t, sampler_t, int2);
float4 read_imagef(image2d_t, sampler_t, float2);
void write_imageui(image2d_t, int2, uint4);
void write_imageui(image2d_t, int2, uint);
void write_imagei(image2d_t, int2, int4);
void write_imagef(image2d_t, int2, float4);
void write_imagef(image2d_t, int2, float);

int get_global_id(int);
int get_global_size(int);
int get_image_width(image2d_t);
int get_image_height(image2d_t);

int min(int, int);
int max(int, int);
int abs_diff(int, int);
bool isnan(float);

float native_sqrt(float);
float4 native_sqrt(float4);
char convert_char_sat(float);
uchar convert_uchar_sat(float);
int4 convert_int4(uint4);
uint4 convert_uint4(float4);
int convert_int(uchar);
float2 convert_float2(int2);
float4 convert_float4(int4);
float4 convert_float4(uint4);
int abs(int);
float fabs(float);
float round(float);

int atomic_inc(volatile global int*);

#define INFINITY 9999999999999.9f
#define NAN 9999999999999.9f
