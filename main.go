package main;
import(
	"github.com/BurntSushi/toml"
	"fmt"
	"os"
        "crypto/sha256"
	"log"
	"io"
	"bytes"
)

type (
	tCONFIG struct {
		Name string
		Bootable bool
		Immutable bool
		Bind_mounts []string
		Sync_paths []string
		Create_dirs []string
	}


)





func pathExists(path string) bool {
	fInfo,err := os.Stat(path)
        if err != nil {
		return false;
        }
	return fInfo != nil
}



func bindMounts(conf tCONFIG){

}

func copyPaths(conf tCONFIG){

}

// switches to the root
func activate(conf tCONFIG){

}

func hashFile(path string) ([]byte,error) {
  file, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
    return nil,err
  }
  defer file.Close()

  hash := sha256.New()
  if _, err := io.Copy(hash, file); err != nil {
    log.Fatal(err)
    return nil, err
  }

  return hash.Sum(nil),nil
  
}

func createChroot(name string) {
	if !pathExists("/etc/noix") {
		err := os.MkdirAll("/etc/noix", os.ModePerm)
		if err != nil {
			fmt.Println("error in createChroot: mkdir -p /etc/noix missing permissions to create /etc/noix")
		}
	}
        path := fmt.Sprintf("/etc/noix/%s", name)
	err := os.Mkdir(path,os.ModePerm)
	if err != nil {
		fmt.Printf("error in createChroot: Failed to create directory %s\n", path);
	}
}



func main(){
   if len(os.Args) < 3 {
	return
   }
   
   if os.Args[1] == "build" || os.Args[1] == "-b" {
	var config tCONFIG
	_,_ = toml.DecodeFile(os.Args[2],&config)
   	createChroot(config.Name)
	sum,_ := hashFile(os.Args[2])
        sum1,_ := hashFile(os.Args[2])
	fmt.Println(bytes.Compare(sum,sum1))
	

   }
}
