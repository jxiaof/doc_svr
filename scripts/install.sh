#!/bin/bash
# Go 1.26.3 自动安装脚本（Ubuntu 全局持久化版）

# 1. 定义版本（可自行修改）
GO_VERSION="1.26.3"
GO_FILE="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_FILE}"

# 2. 下载 Go 安装包（已存在则跳过）
echo "===== 开始下载 Go ${GO_VERSION} ====="
if [ ! -f "${GO_FILE}" ]; then
    wget ${GO_URL}
else
    echo "安装包已存在，跳过下载"
fi

# 3. 清理旧版本 Go
echo "===== 清理旧版本 Go ====="
sudo rm -rf /usr/local/go

# 4. 解压到官方推荐目录 /usr/local
echo "===== 解压安装 Go ====="
sudo tar -C /usr/local -xzf ${GO_FILE}

# 5. 永久配置环境变量（写入系统全局配置 /etc/profile，所有用户+重启永久生效）
echo "===== 配置永久环境变量 ====="
PROFILE_PATH="/etc/profile"
# 避免重复写入
if ! grep -q "/usr/local/go/bin" ${PROFILE_PATH}; then
    echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a ${PROFILE_PATH}
    echo 'export GOPATH=$HOME/go' | sudo tee -a ${PROFILE_PATH}
    echo 'export PATH=$PATH:$GOPATH/bin' | sudo tee -a ${PROFILE_PATH}
fi

# 6. 立即生效环境变量（当前终端直接用）
source ${PROFILE_PATH}

# 7. 配置 Go 模块、国内代理、校验库（永久生效）
echo "===== 配置 Go 国内代理 ====="
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn

# 8. 验证安装结果
echo "===== 安装完成，验证版本 ====="
go version
go env | grep -E "GOPROXY|GO111MODULE|GOROOT"

echo -e "\n✅ Go ${GO_VERSION} 安装成功！环境变量已永久持久化，重启系统依然有效！"