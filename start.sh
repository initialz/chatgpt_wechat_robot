# 这是一个bash脚本，用于启动微信机器人，主要有以下几个需求点
# 1. 启动机器人的命令是：go run main.go
# 2. 机器人需要在后台运行，用nohup命令，并且将标准输出定向到/tmp/chat.log
# 3. 机器人启动以后需要对进程进行监控，如果发现进程退出了，就立刻启动一个新的进程，并继续监控
# 4. 监控时间间隔为5秒

#！/bin/bash

# 无限循环
while true
do
  # 检查机器人进程是否存在
  if ps -ef | grep "go run main.go" | grep -v grep > /dev/null
  then
    # 进程存在，什么都不做
    :
  else
    # 进程不存在，启动机器人
    nohup go run main.go &> /tmp/chat.log &
  fi
  # 睡眠5秒
  sleep 5
done
