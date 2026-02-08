package perf

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println("=== Buildinfo ===")
		fmt.Printf("GoVersion: %s\n", info.GoVersion)
		fmt.Printf("ModulePath: %s\n", info.Path)
		fmt.Printf("ModuleVersion: %s\n", info.Main.Version)

		// 输出所有构建设置
		fmt.Println("\n=== BuildSettings ===")
		for _, setting := range info.Settings {
			fmt.Printf("%s = %s\n", setting.Key, setting.Value)
		}
	}
}
