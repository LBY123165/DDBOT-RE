# Makefile for DDBOT-RE

# 构建目标
target = ddbot.exe

# 源文件
main = ./cmd/main.go

# 构建命令
build:
	@echo "构建DDBOT..."
	@go build -o $(target) $(main)
	@echo "构建完成: $(target)"

# 运行命令
run:
	@echo "运行DDBOT..."
	@go run $(main)

# 清理命令
clean:
	@echo "清理构建文件..."
	@if exist $(target) del $(target)
	@echo "清理完成"

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@echo "依赖安装完成"

# 测试模块
test-module:
	@echo "测试模块管理系统..."
	@go run test_module.go

# 帮助信息
help:
	@echo "DDBOT-RE 构建工具"
	@echo "可用命令:"
	@echo "  make build   - 构建DDBOT"
	@echo "  make run     - 运行DDBOT"
	@echo "  make clean   - 清理构建文件"
	@echo "  make deps    - 安装依赖"
	@echo "  make test-module - 测试模块管理系统"
	@echo "  make help    - 显示帮助信息"

.PHONY: build run clean deps test-module help
