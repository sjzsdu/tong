#!/bin/bash

# 定义 GitHub 仓库信息
REPO="sjzsdu/tong"
API_URL="https://api.github.com/repos/$REPO/releases/latest"

# 获取最新版本号
VERSION=$(curl -s $API_URL | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# 检测操作系统类型
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="mac"
else
    echo "不支持的操作系统类型: $OSTYPE"
    exit 1
fi

# 构建下载 URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/tong-$OS"

# 下载文件
echo "正在下载 tong-$OS ..."
curl -L $DOWNLOAD_URL -o tong

# 使文件可执行
chmod +x tong

# 确定安装目录
INSTALL_DIR="/usr/local/bin"
if [[ ! -w "$INSTALL_DIR" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# 移动文件到安装目录
echo "正在安装 tong 到 $INSTALL_DIR ..."
mv tong "$INSTALL_DIR/tong"

# 检查安装目录是否在 PATH 中
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "请将 $INSTALL_DIR 添加到你的 PATH 环境变量中。"
    echo "你可以通过在 ~/.bashrc 或 ~/.zshrc 中添加以下行来实现："
    echo "export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "tong 安装完成！"