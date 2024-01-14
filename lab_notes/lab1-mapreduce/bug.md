# Lab1 bugs

- rpc server 和 coordinator是并发执行，对共享数据需要加锁
- 打开文件
  - 使用os.Open() 打开的文件只有读的权限
  - 使用os.Openfile()可以指定，读写，操作方式(append, create, trunc)等
- 不要在死循环里使用 defer
- 

go语言 字符串和数字转换 

### 7.8

1. worker 传给rpc的参数结构体，初始化之后立刻使用。不能复用

```c++
// args := MrArgs{}
	// reply := MrReply{}  
	for {
		args := MrArgs{}
		reply := MrReply{}  // rpc参数不能复用
		ok := call(taskRequest, &args, &reply)
		if !ok {
			break
		}
    ....
  }
```

2. Crash test中，worker有0.33的概率crash或者陷入较长时间的沉睡。假如一个worker沉睡时间过长，让coordinator重新分配map任务，这样就有多个worker处理相同的map任务，向中间文件写相同数据??

   - 一种解决方法，给map func reduce func设置执行时间。如果超过了这个时间，本次任务作废（默认coordinator已经重新分配任务)，进行下一次任务请求

3. 如何避免worker沉睡时间过长，coordinator重新分配这个任务，导致两个worker执行相同任务，即写同一份数据(map写中间文件，reduce写结果文件)

   - 分配一个taskid，相同的map任务taskid相同，将结果写到temp-taskid-0,  temp-taskid-1,...中间文件。使用os.Rename()将中间文件重命名为mr-taskid-y。最终只有一份数据
   - reduce任务的taskid表示输入文件的编号，这些文件分散为mr-k-taskid，k表示所有的map任务编号。这样相当于map写矩阵的行，reduce读矩阵的列
   - 额外的方法，延长coordiantor等待时间

4. Job-count测试

   ```go
   var count int
   
   func Map(filename string, contents string) []mr.KeyValue {
   	me := os.Getpid()
   	f := fmt.Sprintf("mr-worker-jobcount-%d-%d", me, count)
   	count++
   	err := ioutil.WriteFile(f, []byte("x"), 0666)
   	if err != nil {
   		panic(err)
   	}
   	time.Sleep(time.Duration(2000+rand.Intn(3000)) * time.Millisecond)
   	return []mr.KeyValue{mr.KeyValue{"a", "x"}}
   }
   
   func Reduce(key string, values []string) string {
   	files, err := ioutil.ReadDir(".")
   	if err != nil {
   		panic(err)
   	}
   	invocations := 0
   	for _, f := range files {
   		if strings.HasPrefix(f.Name(), "mr-worker-jobcount") {
   			invocations++
   		}
   	}
   	return strconv.Itoa(invocations)
   }
   ```

   

5. rpc是否要考虑网络延迟



![image-20230710011734085](/Users/leslie/Library/Application Support/typora-user-images/image-20230710011734085.png)
