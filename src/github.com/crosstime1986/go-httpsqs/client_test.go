package httpsqs

import (
	"testing"
	"httpsqs"
)

var IP string = "192.168.1.20"
var val string = "习近平指出，近期我们在上海和巴西福塔莱萨两次会晤，达成一系列重要合作共识，两国政府各部门和各地方正在积极落实，中俄关系和各领域合作势头强劲。本月初，你亲自出席中俄东线天然气管道俄罗斯境内段开工仪式，体现了你对两国能源合作的重视，对深化双方各领域务实合作起到了带动和示范作用。目前双方正在积极探讨高铁合作，卫星导航系统合作已经启动，联合研制远程宽体客机和重型直升机等大项目合作又取得新进展"

func BenchmarkPuts(b *testing.B) {
//	b.StopTimer()
	client := httpsqs.NewClient(IP, 1218, "", false)
//	b.StartTimer()
	for i := 0; i < b.N; i++ {
		client.Puts("test", val)
	}
}

func BenchmarkGets(b *testing.B) {
	b.StopTimer()
	client := httpsqs.NewClient(IP, 1218, "", false)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		client.Gets("test")
	}
}

func BenchmarkPPuts(b *testing.B) {
	//	b.StopTimer()
	client := httpsqs.NewClient(IP, 1218, "", false)
	//	b.StartTimer()
	for i := 0; i < b.N; i++ {
		client.PPuts("test", val)
	}
}

func BenchmarkPGets(b *testing.B) {
	b.StopTimer()
	client := httpsqs.NewClient(IP, 1218, "", false)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		client.PGets("test")
	}
}




