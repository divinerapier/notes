# INTRODUCE TO SEAWEEDFS

## 概述

Seaweedfs 上传文件

### Assign

`Master Server` 提供 `grpc` 接口 `Assign`, 根据请求参数, 从一个可写`Volume`分配出一个 `File ID`

``` protobuf3
// TODO: Assign function signature

```

如果当前没有可写`Volume`时，并且`Volume` 数量不足最大值时`(ms.Topo.FreeSpace() > 0)`, 会尝试扩容

``` go
// Master Server
func (vg *VolumeGrowth) AutomaticGrowByType(option *VolumeGrowOption, grpcDialOption grpc.DialOption, topo *Topology) (count int, err error) {
    return vg.GrowByCountAndType(grpcDialOption, vg.findVolumeCount(option.ReplicaPlacement.GetCopyCount()), option, topo)
}
func (vg *VolumeGrowth) GrowByCountAndType(grpcDialOption grpc.DialOption, targetCount int, option *VolumeGrowOption, topo *Topology) (counter int, err error) {
    for i := 0; i < targetCount; i++ {
        if c, e := vg.findAndGrow(grpcDialOption, topo, option); e == nil {
            counter += c
        } else {
            return counter, e
        }
    }
    return
}
func (vg *VolumeGrowth) findAndGrow(grpcDialOption grpc.DialOption, topo *Topology, option *VolumeGrowOption) (int, error) {
    servers, e := vg.findEmptySlotsForOneVolume(topo, option)
    if e != nil {
        return 0, e
    }
    vid, raftErr := topo.NextVolumeId()
    if raftErr != nil {
        return 0, raftErr
    }
    err := vg.grow(grpcDialOption, topo, vid, option, servers...)
    return len(servers), err
}

// 目前不清楚这个函数的作用
func (vg *VolumeGrowth) findEmptySlotsForOneVolume(topo *Topology, option *VolumeGrowOption) (servers []*DataNode, err error) {
    return
}

func (vg *VolumeGrowth) grow(grpcDialOption grpc.DialOption, topo *Topology, vid storage.VolumeId, option *VolumeGrowOption, servers ...*DataNode) error {
    for _, server := range servers {
        // 调用 volume service 分配 volume
        if err := AllocateVolume(server, grpcDialOption, vid, option); err == nil {
            vi := storage.VolumeInfo{
                Id:               vid,
                Size:             0,
                Collection:       option.Collection,
                ReplicaPlacement: option.ReplicaPlacement,
                Ttl:              option.Ttl,
                Version:          storage.CurrentVersion,
            }
            server.AddOrUpdateVolume(vi)
            topo.RegisterVolumeLayout(vi, server)
            glog.V(0).Infoln("Created Volume", vid, "on", server.NodeImpl.String())
        } else {
            glog.V(0).Infoln("Failed to assign volume", vid, "to", servers, "error", err)
            return fmt.Errorf("Failed to assign %d: %v", vid, err)
        }
    }
    return nil
}

// 神奇的函数，输入 ReplicaPlacement.GetCopyCount() 得到volume数量
func (vg *VolumeGrowth) findVolumeCount(copyCount int) (count int) {
    switch copyCount {
    case 1:
        count = 7
    case 2:
        count = 6
    case 3:
        count = 3
    default:
        count = 1
    }
    return
}

```

``` go
// Volume Server

// Volume Server grpc api
func (vs *VolumeServer) AllocateVolume(ctx context.Context, req *volume_server_pb.AllocateVolumeRequest) (*volume_server_pb.AllocateVolumeResponse, error) {

    resp := &volume_server_pb.AllocateVolumeResponse{}

    err := vs.store.AddVolume(
        storage.VolumeId(req.VolumeId),
        req.Collection,
        vs.needleMapKind,
        req.Replication,
        req.Ttl,
        req.Preallocate,
    )

    return resp, err
}

func (s *Store) AddVolume(volumeId VolumeId, collection string, needleMapKind NeedleMapType, replicaPlacement string, ttlString string, preallocate int64) error {
    // replicaPlacement 是一个长度为3的字符串
    // 每一位都是一个 0-9 的数字
    rt, e := NewReplicaPlacementFromString(replicaPlacement)
    if e != nil {
        return e
    }
    ttl, e := ReadTTL(ttlString)
    if e != nil {
        return e
    }
    e = s.addVolume(volumeId, collection, needleMapKind, rt, ttl, preallocate)
    return e
}

func (s *Store) addVolume(vid VolumeId, collection string, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) error {
    if s.findVolume(vid) != nil {
        return fmt.Errorf("Volume Id %d already exists!", vid)
    }
    // 找到 (location.MaxVolumeCount - location.VolumesLen()) 值最大的 Location
    if location := s.FindFreeLocation(); location != nil {
        if volume, err := NewVolume(location.Directory, collection, vid, needleMapKind, replicaPlacement, ttl, preallocate); err == nil {
            location.SetVolume(vid, volume)
            s.NewVolumeIdChan <- vid
            return nil
        } else {
            return err
        }
    }
    return fmt.Errorf("No more free space left")
}

func (s *Store) findVolume(vid VolumeId) *Volume {
    for _, location := range s.Locations {
        if v, found := location.FindVolume(vid); found {
            return v
        }
    }
    return nil
}

func NewVolume(dirname string, collection string, id VolumeId, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) (v *Volume, e error) {
    // if replicaPlacement is nil, the superblock will be loaded from disk
    v = &Volume{dir: dirname, Collection: collection, Id: id}
    v.SuperBlock = SuperBlock{ReplicaPlacement: replicaPlacement, Ttl: ttl}
    v.needleMapKind = needleMapKind
    e = v.load(true, true, needleMapKind, preallocate)
    return
}

func (v *Volume) load(alsoLoadIndex bool, createDatIfMissing bool, needleMapKind NeedleMapType, preallocate int64) error {
    var e error
    fileName := v.FileName()
    alreadyHasSuperBlock := false

    exists, canRead, canWrite, modifiedTime, fileSize := checkFile(fileName + ".dat")
    alreadyHasSuperBlock = exists && fileSize >= _SuperBlockSize
    if !exists {
        if !createDatIfMissing {
            return fmt.Errorf("Volume Data file %s.dat does not exist.", fileName)
        }
        v.dataFile, e = createVolumeFile(fileName+".dat", preallocate)
        if e != nil {
            return e
        }
    }

    if alreadyHasSuperBlock {
        e = v.readSuperBlock()
    } else {
        e = v.maybeWriteSuperBlock()
    }
    if e != nil || alsoLoadIndex {
        return e
    }
    var indexFile *os.File
    if v.readOnly {
        indexFile, e = os.OpenFile(fileName+".idx", os.O_RDONLY, 0644)
        if e != nil {
            return e
        }
    } else {
        indexFile, e = os.OpenFile(fileName+".idx", os.O_RDWR|os.O_CREATE, 0644)
        if e != nil {
            return e
        }
    }
    v.readOnly = CheckVolumeDataIntegrity(v, indexFile) != nil
    switch needleMapKind {
    case NeedleMapInMemory:
        v.nm, e = LoadCompactNeedleMap(indexFile)
    }
    return e
}
```

## 数据结构

### Volume

``` go

type Volume struct {
    SuperBlock SuperBlock
    Needles    []Needle
}

```

#### SuperBlock

参考 `volume_super_block.go:ReadSuperBlock`

``` go
type SuperBlock struct {
    Header [8]byte
}

┌-------------------------------------------------------------------------------┐
| 0       | 1       | 2                 | 4                 | 6                 |
| version | rp      | ttl               | cr                | es                |
└-------------------------------------------------------------------------------┘
rp: replica placement
cr: compact revision
es: extra size
```

#### Needle

##### Index

参考 `volume_read_write.go:(*Volume).writeNeedle`, `volume_read_write.go:(*Volume).readNeedle`

以 `Version3` 为🌰

``` go
type NeedleValue struct {
    Key    NeedleId
    Offset Offset `comment:"Volume offset"` //since aligned to 8 bytes, range is 4G*8=32G
    Size   uint32 `comment:"Size of the data portion"`
}
type Offset struct {
    OffsetHigher
    OffsetLower
}

```

##### Data

``` go
R: required, O: optional
----------┬-------------------------------------------------------------------------------┐
R/O|Offset|    0    |    1    |    2    |    3    |    4    |    5    |    6    |    7    |
----------┼-------------------------------------------------------------------------------┤
R  | 0    | cookie                                | needle id part 0                      |
R  | 8    | needle id part 1                      | size                                  |
----------┼-------------------------------------------------------------------------------┤
O  |16    | data_size: len(needle.data)           | needle.data                           |
O  |      | flag    | some fields                                                         |
O  |      | current_timestamp                                                             |
O  |      | checksum                              | padding                               |
----------┴-------------------------------------------------------------------------------┘

size: 数据部分总长度，及 optional 部分

some fields: [name, mime, last_modified_date, ttl, pairs]

注意:
    如果, len(data) = 0, 则 size = 0;
    否则, size = len(data) +
                (1)(flag) +
                (4)(data_size) +
                (1+name_size)(if name exists) +
                (1+mime_size)(if mime exists) +
                (8)(if last modified date exists) + /*使用后5个字节*/
                (2)(if ttl exists) +
                (2+pairs)(if pairs exists) +
                (4)(checksum) +
                (8)(current time stamp) +
                (n)(padding, 8Bytes alignment)
```

### Filer

#### File ID
