package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/iovxw/downloader"
	"github.com/parnurzeal/gorequest"
	"github.com/wetor/AnimeGo/pkg/cache"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	BangumiArchiveRelease = "https://api.github.com/repos/bangumi/Archive/releases/latest"
	SubjectBucket         = "bangumi_sub"
	SubjectDB             = "bolt_sub.db"
	EpisodeBucket         = "bangumi_ep"
	EpisodeDB             = "bolt_ep.db"
)

type GithubRelease struct {
	Assets []struct {
		Name               string    `json:"name"`
		Size               int64     `json:"size"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadUrl string    `json:"browser_download_url"`
	} `json:"assets"`
}

var (
	SubjectMap   map[int]*Entity
	EpisodeMap   map[int][]*Ep
	SubjectIndex []int
)

func main() {
	fmt.Println("--------------------------------")
	fmt.Println("获取bangumi最新下载地址")
	release := GithubRelease{}
	req := gorequest.New()
	_, _, errs := req.Get(BangumiArchiveRelease).EndStruct(&release)
	if errs != nil {
		fmt.Println(errs)
		return
	}
	fmt.Println(release)
	if len(release.Assets) == 0 {
		fmt.Println("未找到release")
		return
	}
	fmt.Println("--------------------------------")
	fmt.Println("下载bangumi数据")

	sort.Slice(release.Assets, func(i, j int) bool {
		return release.Assets[i].UpdatedAt.After(release.Assets[j].UpdatedAt)
	})
	filename := release.Assets[0].Name
	result := download(release.Assets[0].BrowserDownloadUrl, release.Assets[0].Size, filename)
	if !result {
		fmt.Println("下载失败")
		return
	}
	fmt.Println("--------------------------------")
	fmt.Println("解压bangumi数据")
	time.Sleep(1 * time.Second)

	err := UnZip(".", filename)
	if err != nil {
		fmt.Println("解压失败", err)
		return
	}
	fmt.Println("--------------------------------")
	fmt.Println("清洗bangumi数据")
	time.Sleep(1 * time.Second)
	result = CleanSubject("subject.jsonlines")
	if !result {
		fmt.Println("清洗Subject失败")
		return
	}
	result = CleanEpisode("episode.jsonlines")
	if !result {
		fmt.Println("清洗Episode失败")
		return
	}
	UpdateSubject()
	fmt.Println("--------------------------------")
	fmt.Println("保存bangumi数据到", SubjectDB)
	result = SaveSubjectBolt(SubjectDB)
	if !result {
		fmt.Println("保存到bolt失败")
		return
	}
	fmt.Println("--------------------------------")
	fmt.Println("保存bangumi的ep数据到", EpisodeDB)
	result = SaveEpisodeBolt(EpisodeDB)
	if !result {
		fmt.Println("保存到bolt失败")
		return
	}
	fmt.Println("--------------------------------")
	fmt.Println("数据处理完成！")
}

func download(uri string, size int64, save string) bool {
	start := time.Now()

	file, err := os.Create(save)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer file.Close()
	fileDl, err := downloader.NewFileDl(uri, file, size)
	if err != nil {
		fmt.Println(err)
		return false
	}
	var exit = make(chan bool)
	var resume = make(chan bool)
	var pause bool
	var wg sync.WaitGroup
	wg.Add(1)
	fileDl.OnStart(func() {
		fmt.Println("开始下载：", uri)
		for {
			select {
			case <-exit:
				fmt.Println("下载完成：", save)
				wg.Done()
			default:
				if !pause {
					time.Sleep(time.Second * 1)
				} else {
					<-resume
					pause = false
				}
			}
		}
	})

	fileDl.OnPause(func() {
		pause = true
	})

	fileDl.OnResume(func() {
		resume <- true
	})

	fileDl.OnFinish(func() {
		exit <- true
	})

	fileDl.OnError(func(errCode int, err error) {
		fmt.Println(errCode, err)
	})

	fileDl.Start()
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return true
}

func UnZip(dst, src string) (err error) {
	start := time.Now()
	zr, err := zip.OpenReader(src)
	defer zr.Close()
	if err != nil {
		return
	}
	if dst != "" {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
	}

	// 遍历 zr ，将文件写入到磁盘
	for _, file := range zr.File {
		if file.Name != "subject.jsonlines" && file.Name != "episode.jsonlines" {
			continue
		}
		fmt.Println("解压文件：", file.Name)
		path := filepath.Join(dst, file.Name)

		// 如果是目录，就创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// 获取到 Reader
		fr, err := file.Open()
		if err != nil {
			return err
		}

		// 创建要写出的文件对应的 Write
		fw, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		n, err := io.Copy(fw, fr)
		if err != nil {
			return err
		}

		// 将解压的结果输出
		fmt.Printf("成功解压 %s ，共写入了 %d 个字符的数据\n", path, n)

		fw.Close()
		fr.Close()
	}
	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return nil
}

func CleanSubject(src string) bool {
	start := time.Now()
	SubjectMap = make(map[int]*Entity)
	SubjectIndex = make([]int, 0, 128*1024)
	err := ReadFile(src, func(s string) {
		sub := &Entity{}
		err := json.Unmarshal([]byte(s), sub)
		if err != nil {
			fmt.Println("失败，", err, s)
			return
		}
		if sub.Type != 2 { // 动画
			return
		}
		SubjectMap[sub.ID] = sub
		SubjectIndex = append(SubjectIndex, sub.ID)
	})
	if err != nil {
		return false
	}
	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return true
}

func CleanEpisode(src string) bool {
	start := time.Now()
	EpisodeMap = make(map[int][]*Ep)
	err := ReadFile(src, func(s string) {
		ep := &Ep{}
		err := json.Unmarshal([]byte(s), ep)
		if err != nil {
			fmt.Println("失败，", err, s)
			return
		}
		if ep.Type != 0 { // 普通剧集
			return
		}
		eps, ok := EpisodeMap[ep.SubjectID]
		if !ok {
			eps = make([]*Ep, 0, 16)
		}
		eps = append(eps, ep)
		EpisodeMap[ep.SubjectID] = eps
	})
	if err != nil {
		return false
	}
	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return true
}

func ReadFile(filePath string, handle func(string)) error {
	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		return err
	}
	scan := bufio.NewScanner(f)

	for {
		if !scan.Scan() {
			break
		}
		line := scan.Text()
		line = strings.TrimSpace(line)
		handle(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func UpdateSubject() {
	for _, id := range SubjectIndex {
		eps, ok := EpisodeMap[id]
		if !ok {
			continue
		}
		sub := SubjectMap[id]
		sub.Eps = len(eps)
		if sub.Eps > 0 {
			// TODO: 默认第一个为第一集，可能需要排序操作
			sub.AirDate = eps[0].AirDate
		}
	}
}

// SaveSubjectBolt 保存subject信息的数据
func SaveSubjectBolt(dst string) bool {
	start := time.Now()
	db := cache.NewBolt()
	db.Open(dst)
	defer db.Close()
	keys := make([]interface{}, len(SubjectIndex))
	vals := make([]interface{}, len(SubjectIndex))
	for i, id := range SubjectIndex {
		keys[i] = id
		vals[i] = SubjectMap[id]
	}
	db.Add(SubjectBucket)
	db.BatchPut(SubjectBucket, keys, vals, 0)
	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return true
}

// SaveEpisodeBolt 保存ep信息的数据
func SaveEpisodeBolt(dst string) bool {
	start := time.Now()

	db := cache.NewBolt()
	db.Open(dst)
	defer db.Close()
	keys := make([]interface{}, len(SubjectIndex))
	vals := make([]interface{}, len(SubjectIndex))
	for i, id := range SubjectIndex {
		keys[i] = id
		vals[i] = EpisodeMap[id]
	}
	db.Add(EpisodeBucket)
	db.BatchPut(EpisodeBucket, keys, vals, 0)
	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)
	return true
}
