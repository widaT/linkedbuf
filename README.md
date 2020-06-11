# linkedbuf

一种高效低损耗的buffer，针对go reactor模式下的一种buffer，目标是做到no copy，但是夸block的拼数据包还是有copy。