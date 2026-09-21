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
#include "kdtree.h"

void KDTree::insert(Match* iData) {
	size++;
	std::unique_ptr<KDTree>& side = iData->pos[dim] < data->pos[dim] ? left : right;
	if(side != nullptr) {
		side->insert(iData);
		return;
	}

	side = std::make_unique<KDTree>((dim+1) % 2, iData);
}

void KDTree::rangeSearch(std::vector<Match*>& values, const Eigen::Vector2f& point, const float radius) const {
	if((data->pos - point).norm() <= radius)
		values.push_back(data);

	if(left != nullptr && point[dim] <= data->pos[dim] + radius)
		left->rangeSearch(values, point, radius);
	if(right != nullptr && point[dim] >= data->pos[dim] - radius)
		right->rangeSearch(values, point, radius);
}
