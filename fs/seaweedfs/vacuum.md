# Vacuum

`curl http://master:9333/vol/vacuum?garbageThreshold=0.001` 是 `seaweedfs` 的垃圾回收接口。

由 `volume server` 的4个接口共同提供服务。

``` go
type VolumeServerServer interface {
    VacuumVolumeCheck(VacuumVolumeCheckRequest) VacuumVolumeCheckResponse
    VacuumVolumeCompact(VacuumVolumeCompactRequest) VacuumVolumeCompactResponse
    VacuumVolumeCommit(VacuumVolumeCommitRequest) VacuumVolumeCommitResponse
    VacuumVolumeCleanup(VacuumVolumeCleanupRequest) VacuumVolumeCleanupResponse
}
```

``` go
// VacuumVolumeCheck
// check 的主要逻辑, 计算目标volume被删除大小占总大小的比例,并与 gc 阈值比较(默认0.3)
//
// 由此可见，需要重启 volume 来更新被删除信息, 才能 gc ?
//
func (v *Volume) garbageLevel() float64 {
    if v.ContentSize() == 0 {
        return 0
    }
    return float64(v.nm.DeletedSize()) / float64(v.ContentSize())
}

```
