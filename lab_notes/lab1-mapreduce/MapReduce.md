MapReduce

主从通信， client-server



Worker (client) 向 master (server)发起分配任务请求

- master 查找是否有任务待完成
  - 作业的状态
    - 没有被分配
    - 已经分配，还没有被完成，查看时间是否超过10分钟
    - 已经完成
  
  - master先查找 map任务，再查找 reduce 任务
    - map任务都完成了 才能分配reduce任务
    - 在查找map任务时，记录是否有map任务正在进行
    - 如果map任务在进行，那么告知client需要等待
      - 通过reply.Op
  

Worker接收任务，通过参数得到任务的名称reduce or map

- 读取相应的文件 进行map or reduce
- 向master传达**任务完成信息**
  - 完成的任务类型
  - map的文件或者reduce的编号


Server 收到任务完成信息 进行处理

- 标记状态为完成

- 如果是reduce task 额外处理

启动master的进程要检查任务是否完成

- 调用done函数，done函数检查reduce task任务是否都完成



![image-20230708173415618](/Users/leslie/Library/Application Support/typora-user-images/image-20230708173415618.png)

