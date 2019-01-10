package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"time"

	"image/draw"

	"github.com/nfnt/resize"

	"io/ioutil"
	"os"
	"sync"

	"github.com/golang/freetype"

	"github.com/tealeg/xlsx"
)

const (
	bigWidth  = 480
	bigHeight = 600

	bgWidth  = 170 //背景宽度
	bgHeight = 280 //背景高度

	picWidth  = 150 // 图片的大小 宽度
	picHeight = 220 // 图片的大小 高度

	marginLeft = 10
	marginTop  = 10

	fontFile = "/Applications/Microsoft Word.app/Contents/Resources/Fonts/SimHei.ttf" // 需要使用的字体文件
	fontSize = 3                                                                      // 字体尺寸
	fontDPI  = 360                                                                    // 屏幕每英寸的分辨率

	excelFile = "/Users/yangzuwei/Desktop/学生.xlsx"
	dest      = "/Users/yangzuwei/Desktop/res/"
)

type student struct {
	id      string
	name    string
	aux_num string
	path    string
}

var studentPath map[string]string

var wg sync.WaitGroup

var mutex sync.Mutex

func main() {
	t := time.Now()

	c := initFont()
	os.Mkdir(dest, 0777)
	paths := scanAll(dest)
	//fmt.Println(paths)
	//初始化数据
	initData := initStudents(excelFile, paths)
	bg := image.NewNRGBA(image.Rect(0, 0, bgWidth, bgHeight))
	//draw.Draw(&bg, (&bg).Bounds(), image.White, image.ZP, draw.Src)

	for _, v := range initData {
		wg.Add(1)
		bg := bg
		go drawString(v, c, bg)
	}
	wg.Wait()
	elapsed := time.Since(t)
	fmt.Println("total time is ", elapsed)
}

func drawString(studentInfo student, c *freetype.Context, bg *image.NRGBA) {
	destFile := dest + studentInfo.id + ".jpg"
	//os.Remove(destFile)
	imgfile, _ := os.Create(destFile)
	src := "../sample.jpg" //原始文件路径 studentInfo.path

	mutex.Lock()
	draw.Draw(bg, bg.Bounds(), image.White, image.ZP, draw.Src)
	copyImageToBg(src, bg)
	drawFontOnImage(studentInfo, bg, c)
	jpeg.Encode(imgfile, bg, &jpeg.Options{Quality: 90})
	imgfile.Close()
	mutex.Unlock()

	wg.Done()
}

func drawText(studentInfo student, c *freetype.Context) {
	bg := image.NewNRGBA(image.Rect(0, 0, bgWidth, bgHeight))
	destFile := dest + studentInfo.id + ".jpg"
	//os.Remove(destFile)
	imgfile, _ := os.Create(destFile)
	src := "sample.jpg" //原始文件路径 studentInfo.path
	drawFontOnImage(studentInfo, bg, c)
	copyImageToBg(src, bg)
	jpeg.Encode(imgfile, bg, &jpeg.Options{Quality: 90})
	imgfile.Close()
	wg.Done()
}

func copyImageToBg(src string, rgba *image.NRGBA) {
	fd, _ := os.Open(src)
	srcImage, _, _ := image.Decode(fd)
	srcImage = resize.Resize(picWidth, picHeight, srcImage, resize.Lanczos3) //重新设置原始图片的宽高
	draw.Draw(rgba, image.Rectangle{image.Point{marginLeft, marginTop}, image.Point{marginLeft + picWidth, marginTop + picHeight}}, srcImage, image.ZP, draw.Src)
}

func initFont() *freetype.Context {
	//设置文字对像
	fontBytes, _ := ioutil.ReadFile(fontFile)
	font, _ := freetype.ParseFont(fontBytes)
	c := freetype.NewContext()
	c.SetDPI(fontDPI)       //分辨率
	c.SetFont(font)         //字符
	c.SetFontSize(fontSize) //大小
	c.SetSrc(image.Black)   //字体颜色
	return c
}

func drawFontOnImage(studentInfo student, bg *image.NRGBA, c *freetype.Context) *image.NRGBA {
	c.SetDst(bg)           //要画进去的图
	c.SetClip(bg.Bounds()) //背景
	var text [3]string = [3]string{studentInfo.id, studentInfo.name, studentInfo.aux_num}
	for rowNum := 1; rowNum < 4; rowNum++ {
		x := (picWidth - len(text[rowNum-1])*6) >> 1
		if rowNum == 2 {
			x = (picWidth - len(text[rowNum-1])*4) >> 1
		}
		y := picHeight + marginTop - 3 + (fontSize*5)*rowNum
		pt := freetype.Pt(x, y)          // 字出现的位置
		c.DrawString(text[rowNum-1], pt) //绘制
	}
	return bg
}

func initStudents(excelFile string, paths map[string]string) map[string]student {

	//读取excel中的学生信息 和 原始文件夹中的信息 组合成为待用的map
	studentInfo := make(map[string]student)
	xlFile, err := xlsx.OpenFile(excelFile)
	if err != nil {
		fmt.Printf("open failed: %s\n", err)
	}
	for _, sheet := range xlFile.Sheets {
		fmt.Println(sheet.Name)
	}
	mySheet := xlFile.Sheets[0] //第一个sheet
	maxRow := mySheet.MaxRow
	for i := 1; i < maxRow; i++ {
		//第1列 id 第2列名字 第3列aux_num
		studentInfo[mySheet.Cell(i, 0).Value] = student{
			id:      mySheet.Cell(i, 0).Value,
			name:    mySheet.Cell(i, 1).Value,
			aux_num: mySheet.Cell(i, 2).Value,
			path:    paths[mySheet.Cell(i, 0).Value],
		}
	}
	return studentInfo
}

/**
 * 读取 excel 文件里面的内容
**/
func readPath(excelFile string) map[string]string {
	studentPath := make(map[string]string)

	xlFile, err := xlsx.OpenFile(excelFile)
	if err != nil {
		fmt.Printf("open failed: %s\n", err)
	}

	for _, sheet := range xlFile.Sheets {
		fmt.Println(sheet.Name)
	}

	mySheet := xlFile.Sheets[0]
	maxRow := mySheet.MaxRow
	for i := 0; i < maxRow; i++ {
		studentPath[mySheet.Cell(i, 0).Value] = mySheet.Cell(i, 1).Value
	}

	return studentPath
}

func scanAll(path string) map[string]string {
	result := make(map[string]string) //结果集
	paths := []string{path}           //目录栈：存储需要遍历的文件夹，初始化时传入需要遍历的文件夹

	for len(paths) > 0 { //目录栈不为空则不断循环
		//出栈pop
		dir := paths[len(paths)-1]
		paths = paths[:len(paths)-1]

		//遍历pop出的的文件夹
		files, _ := ioutil.ReadDir(dir)
		for _, f := range files {
			p := dir + "/" + f.Name() //拼接路径
			if f.IsDir() {
				paths = append(paths, p) //如果是文件夹类型则入栈
			} else {
				result[f.Name()] = p //如果是文件则存结果
			}
		}
	}
	return result
}
