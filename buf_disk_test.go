package main

import (
	"bufio"
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiskQueue(t *testing.T) {

	Convey("TestDiskQueue", t, func() {
		l := log.New(os.Stderr, "", log.LstdFlags)
		dqName := "test_disk_queue" + strconv.Itoa(int(time.Now().Unix()))
		dq := newDiskQueue(dqName, os.TempDir(), 1024, 2500, 2*time.Second, l)
		So(dq, ShouldNotEqual, nil)
		So(dq.Depth(), ShouldEqual, int64(0))

		msg := []byte("test")
		err := dq.Put(msg)
		So(err, ShouldEqual, nil)
		So(dq.Depth(), ShouldEqual, int64(1))

		msgOut := <-dq.ReadChan()
		So(string(msgOut), ShouldEqual, string(msg))
		dq.Delete()
	})

	Convey("TestDiskQueueRoll", t, func() {
		l := log.New(os.Stderr, "", log.LstdFlags)
		dqName := "test_disk_queue_roll" + strconv.Itoa(int(time.Now().Unix()))
		dq := newDiskQueue(dqName, os.TempDir(), 100, 2500, 2*time.Second, l)
		So(dq, ShouldNotEqual, nil)
		So(dq.Depth(), ShouldEqual, int64(0))

		msg := []byte("aaaaaaaaaa")
		for i := 0; i < 10; i++ {
			err := dq.Put(msg)
			So(err, ShouldEqual, nil)
			So(dq.Depth(), ShouldEqual, int64(i+1))
		}

		So(dq.(*diskQueue).writeFileNum, ShouldEqual, int64(1))
		So(dq.(*diskQueue).writePos, ShouldEqual, int64(28))
		dq.Delete()
	})
}

func TestDiskQueueEmpty(t *testing.T) {
	Convey("TestDiskQueueEmpty", t, func() {
		l := log.New(os.Stderr, "", log.LstdFlags)
		dqName := "test_disk_queue_empty" + strconv.Itoa(int(time.Now().Unix()))
		dq := newDiskQueue(dqName, os.TempDir(), 100, 2500, 2*time.Second, l)
		So(dq, ShouldNotEqual, nil)
		So(dq.Depth(), ShouldEqual, int64(0))

		msg := []byte("aaaaaaaaaa")

		for i := 0; i < 100; i++ {
			err := dq.Put(msg)
			So(err, ShouldEqual, nil)
			So(dq.Depth(), ShouldEqual, int64(i+1))
		}

		for i := 0; i < 3; i++ {
			<-dq.ReadChan()
		}

		for {
			if dq.Depth() == 97 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		So(dq.Depth(), ShouldEqual, int64(97))

		numFiles := dq.(*diskQueue).writeFileNum
		dq.Empty()

		f, err := os.OpenFile(dq.(*diskQueue).metaDataFileName(), os.O_RDONLY, 0600)
		So(f, ShouldEqual, (*os.File)(nil))
		So(os.IsNotExist(err), ShouldEqual, true)
		for i := int64(0); i <= numFiles; i++ {
			f, err := os.OpenFile(dq.(*diskQueue).fileName(i), os.O_RDONLY, 0600)
			So(f, ShouldEqual, (*os.File)(nil))
			So(os.IsNotExist(err), ShouldEqual, true)
		}
		So(dq.Depth(), ShouldEqual, int64(0))
		So(dq.(*diskQueue).readFileNum, ShouldEqual, dq.(*diskQueue).writeFileNum)
		So(dq.(*diskQueue).readPos, ShouldEqual, dq.(*diskQueue).writePos)
		So(dq.(*diskQueue).nextReadPos, ShouldEqual, dq.(*diskQueue).readPos)
		So(dq.(*diskQueue).nextReadFileNum, ShouldEqual, dq.(*diskQueue).readFileNum)
		for i := 0; i < 100; i++ {
			err := dq.Put(msg)
			So(err, ShouldEqual, nil)
			So(dq.Depth(), ShouldEqual, int64(i+1))
		}

		for i := 0; i < 100; i++ {
			<-dq.ReadChan()
		}

		for {
			if dq.Depth() == 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		So(dq.Depth(), ShouldEqual, int64(0))
		So(dq.(*diskQueue).readFileNum, ShouldEqual, dq.(*diskQueue).writeFileNum)
		So(dq.(*diskQueue).readPos, ShouldEqual, dq.(*diskQueue).writePos)
		So(dq.(*diskQueue).nextReadPos, ShouldEqual, dq.(*diskQueue).readPos)
		dq.Delete()
	})
}

func TestDiskQueueCorruption(t *testing.T) {
	Convey("TestDiskQueueCorruption", t, func() {
		l := log.New(os.Stderr, "", log.LstdFlags)
		dqName := "test_disk_queue_corruption" + strconv.Itoa(int(time.Now().Unix()))
		dq := newDiskQueue(dqName, os.TempDir(), 1000, 5, 2*time.Second, l)

		msg := make([]byte, 123)
		for i := 0; i < 25; i++ {
			dq.Put(msg)
		}

		So(dq.Depth(), ShouldEqual, int64(25))

		// corrupt the 2nd file
		dqFn := dq.(*diskQueue).fileName(1)
		os.Truncate(dqFn, 500)

		for i := 0; i < 19; i++ {
			So(string(<-dq.ReadChan()), ShouldEqual, string(msg))
		}

		// corrupt the 4th (current) file
		dqFn = dq.(*diskQueue).fileName(3)
		os.Truncate(dqFn, 100)

		dq.Put(msg)

		So(string(<-dq.ReadChan()), ShouldEqual, string(msg))
		dq.Delete()
	})
}

func TestDiskQueueTorture(t *testing.T) {
	Convey("TestDiskQueueTorture", t, func() {
		var wg sync.WaitGroup

		l := log.New(os.Stderr, "", log.LstdFlags)
		dqName := "test_disk_queue_torture" + strconv.Itoa(int(time.Now().Unix()))
		dq := newDiskQueue(dqName, os.TempDir(), 262144, 2500, 2*time.Second, l)
		So(dq, ShouldNotEqual, nil)
		So(dq.Depth(), ShouldEqual, int64(0))

		msg := []byte("aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeffffffffff")

		numWriters := 4
		numReaders := 4
		readExitChan := make(chan int)
		writeExitChan := make(chan int)

		var depth int64
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					time.Sleep(100000 * time.Nanosecond)
					select {
					case <-writeExitChan:
						return
					default:
						err := dq.Put(msg)
						if err == nil {
							atomic.AddInt64(&depth, 1)
						}
					}
				}
			}()
		}

		time.Sleep(1 * time.Second)

		dq.Close()

		t.Logf("closing writeExitChan")
		close(writeExitChan)
		wg.Wait()

		t.Logf("restarting diskqueue")

		dq = newDiskQueue(dqName, os.TempDir(), 262144, 2500, 2*time.Second, l)
		So(dq, ShouldNotEqual, nil)
		So(dq.Depth(), ShouldEqual, depth)

		var read int64
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					time.Sleep(100000 * time.Nanosecond)
					select {
					case <-dq.ReadChan():
						atomic.AddInt64(&read, 1)
					case <-readExitChan:
						return
					}
				}
			}()
		}

		t.Logf("waiting for depth 0")
		for {
			if dq.Depth() == 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		t.Logf("closing readExitChan")
		close(readExitChan)
		wg.Wait()

		So(read, ShouldEqual, depth)

		dq.Delete()
	})
}

func BenchmarkDiskQueuePut(b *testing.B) {
	b.StopTimer()
	l := log.New(os.Stderr, "", log.LstdFlags)
	dqName := "bench_disk_queue_put" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	dq := newDiskQueue(dqName, os.TempDir(), 1024768*100, 2500, 2*time.Second, l)
	size := 1024
	b.SetBytes(int64(size))
	data := make([]byte, size)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		dq.Put(data)
	}
	dq.Delete()
}

func BenchmarkDiskWrite(b *testing.B) {
	b.StopTimer()
	fileName := "bench_disk_queue_put" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	f, _ := os.OpenFile(path.Join(os.TempDir(), fileName), os.O_RDWR|os.O_CREATE, 0600)
	size := 256
	b.SetBytes(int64(size))
	data := make([]byte, size)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		f.Write(data)
	}
	f.Sync()
	os.Remove(f.Name())
}

func BenchmarkDiskWriteBuffered(b *testing.B) {
	b.StopTimer()
	fileName := "bench_disk_queue_put" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	f, _ := os.OpenFile(path.Join(os.TempDir(), fileName), os.O_RDWR|os.O_CREATE, 0600)
	size := 256
	b.SetBytes(int64(size))
	data := make([]byte, size)
	w := bufio.NewWriterSize(f, 1024*4)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		w.Write(data)
		if i%1024 == 0 {
			w.Flush()
		}
	}
	w.Flush()
	f.Sync()
	os.Remove(f.Name())
}

// this benchmark should be run via:
//    $ go test -test.bench 'DiskQueueGet' -test.benchtime 0.1
// (so that it does not perform too many iterations)
func BenchmarkDiskQueueGet(b *testing.B) {
	b.StopTimer()
	l := log.New(os.Stderr, "", log.LstdFlags)
	dqName := "bench_disk_queue_get" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	dq := newDiskQueue(dqName, os.TempDir(), 1024768, 2500, 2*time.Second, l)
	for i := 0; i < b.N; i++ {
		dq.Put([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	}
	b.StartTimer()
	size := 256
	b.SetBytes(int64(size))

	for i := 0; i < b.N; i++ {
		<-dq.ReadChan()
	}

	dq.Delete()
}
