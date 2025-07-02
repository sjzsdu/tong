#!/bin/bash

# 编译项目
go build .

# 移动编译后的二进制文件到 bin 目录
mv tong /Users/juzhongsun/.local/bin/tong
echo "安装完成！"