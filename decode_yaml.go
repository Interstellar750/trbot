package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AdditionalData struct {
	Voices    []VoicePack
	VoiceErr error

	Udonese    *Udonese
	UdoneseErr error
}
type AdditionalDataPath struct {
	Voice   string
	Udonese string
}

func readAdditionalDatas (paths *AdditionalDataPath) AdditionalData {
	voices, voiceErr := readVoicePackFromPath(paths.Voice)
	udonese, udoneseErr := readUdonese(paths.Udonese, metadatafile_name)
	return AdditionalData{
		Voices: voices,
		VoiceErr: voiceErr,
		Udonese: udonese,
		UdoneseErr: udoneseErr,
	}
}

type VoicePack struct {
	Name string `yaml:"name,omitempty"` // 语音包名称
	Voices []struct {
		ID       string `yaml:"ID,omitempty"`       // 语音 ID
		Title    string `yaml:"Title,omitempty"`    // 行内模式时显示的标题
		Caption  string `yaml:"Caption,omitempty"`  // 发送后在语音下方的文字
		VoiceURL string `yaml:"VoiceURL,omitempty"` // 音频文件网络链接
	} `yaml:"voices,omitempty"`
}

// 读取指定目录下所有结尾为 .yaml 或 .yml 的语音文件
func readVoicePackFromPath(path string) ([]VoicePack, error) {
	var packs []VoicePack

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("No voices dir, create a new one: %s", voice_path)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			file, err := os.Open(path)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			defer file.Close()

			var singlePack VoicePack
			decoder := yaml.NewDecoder(file)
			err = decoder.Decode(&singlePack)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			packs = append(packs, singlePack)
		}
		return nil
	})
	if err != nil { return nil, err }
	
	return packs, nil
}

type Udonese struct {
	Count int           `yaml:"count"`
	List  []UdoneseList `yaml:"list"`
}

type UdoneseList struct {
	Word        string           `yaml:"Word,omitempty"`
	MeaningList []UdoneseMeaning `yaml:"MeaningList,omitempty"`
}

type UdoneseMeaning struct {
	Meaning string `yaml:"Meaning"`
	FromID  int64  `yaml:"FromID,omitempty"`
	Name    string `yaml:"Name,omitempty"`
}

func readUdonese(path, name string) (*Udonese, error) {
	var udonese *Udonese

	file, err := os.Open(path + name)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[Udonese]: Not found database file. Created new one")
		SaveYamlDB(path, name, Udonese{})
		return &Udonese{}, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&udonese)
	if err != nil { 
		if err == io.EOF {
			log.Println("[Udonese]: Udonese list looks empty. now format it")
			SaveYamlDB(path, name, Udonese{})
			return &Udonese{}, nil
		}
		log.Println("(func)readUdonese:", err)
		return &Udonese{}, err
	}
	return udonese, nil
}

func addUdonese(udonese *Udonese, params *UdoneseList) {
	for wordIndex, savedList := range udonese.List {
		if savedList.Word == params.Word {
			log.Printf("发现已存在的词 %s，正在检查是否有新增的意思", savedList.Word)
			for _, newMeaning := range params.MeaningList {
				var isreallynew bool = true
				for _, oldmeanlist := range savedList.MeaningList {
					if newMeaning.Meaning == oldmeanlist.Meaning {
						isreallynew = false
					}
				}
				if isreallynew {
					udonese.List[wordIndex].MeaningList = append(udonese.List[wordIndex].MeaningList, newMeaning)
					log.Printf("正在为 %s 添加 %s 意思", udonese.List[wordIndex].Word, newMeaning.Meaning)
				} else {
					log.Println("存在的意思，跳过", newMeaning)
				}
			}
			return
		}
	}
	log.Printf("发现新的词 %s，正在添加 %v", params.Word, params.MeaningList)
	udonese.List = append(udonese.List, *params)
	udonese.Count++
}
