package alsa

//Go Packages
import(
	"errors"
	"github.com/jcelliott/lumber"
	"runtime"
)

/*
	Types
*/

type AudioData []byte

type AudioStream struct{
	Channels int
	Rate int
	SampleFormat SampleType
	DataStream chan AudioData
}

/*
	Global Variables
*/

//Global logger (Change to adjust output..."Trace" maybe?)
var log = lumber.NewConsoleLogger(lumber.WARN)

/*
	Functions
*/

//Returns channel to send audio streams on and a channel that sends nil on finished
func Init(control <-chan bool) (chan<- AudioStream){
	
	//Create Stream channel
	stream := make(chan AudioStream,1) //Allow buffer of one for overlap
	
	go start(stream,control)

	return stream 
}

//Plays by default..send false on control chan to stop
func start(streamChan <-chan AudioStream, control <-chan bool){
	
	var device alsa_device
	
	//Stream loop...end on close of stream
	STREAM:
	for{
		select {
		case msg,ok := <-control:
			if !handleControlMessage(msg,ok,control,nil){
				log.Trace("Control chan closed")
				break STREAM //Rash but probably safe
			}	
		case stream,ok := <-streamChan:
			//Check if streamChan closed
			if ok == false{
				break STREAM
			}
			
			//Configure ALSA device (will use existing device if possible)
			err := configDevice(&device,&stream)
			if err != nil{
				log.Error("error configuring alsa device")
			}
			
			//Play all data in this stream
			DATA:
			for{
				select {
				case msg,ok := <-control:
					if !handleControlMessage(msg,ok,control,&device){
						log.Trace("Control chan Closed: mid data stream")
						break STREAM //Rash but probably safe
					}
				case data,ok := <-stream.DataStream:
					//Check if data stream closed
					if ok == false{
						log.Trace("Track ended")
						break DATA
					}
		
					//Write data to device
					alsa_write(&device,data)
					runtime.Gosched() //Free other things to work
				}
			}
		}
	}
	
	//Close device after last stream
	alsa_close(device.pcm)
	
	log.Trace("ALSA session ended")
}

//Returns when loop should break or continue (True: continue False: break)
func handleControlMessage(msg,ok bool, control <-chan bool, device *alsa_device)(bool){
	if ok == false{
		return false
	}
	
	//Pause alsa device
	if device != nil{
		alsa_pause(device)
	}
	
	//Stall until play given
	for !msg{
		log.Trace("Waiting for next control send")
		
		msg,ok = <-control //Blocking
		if ok == false{
			return false
		}
	}
	log.Trace("Control message handled")
	
	//Resume alsa device
	if device != nil{
		alsa_play(device)
	}
	
	return true //clear to proceed
}

func configDevice(device *alsa_device, stream *AudioStream)(error){

	//Only make new device if one doesn't exist or for different chan_num or rate value
	if device==nil || device.channels != stream.Channels || device.rate != stream.Rate{
		if device.pcm != nil{
			alsa_close(device.pcm)
		}
		
		device.channels = stream.Channels
		device.rate = stream.Rate
		
		switch stream.SampleFormat{
		case INT16_TYPE:
			device.numBytes = 2
		default:
			device.numBytes = 1
		}
				
		pcm, err := alsa_open("default",stream.Channels,stream.Rate)
		if err != nil{
			return errors.New("Error on open")
		}
		
		device.pcm = pcm
	}
	
	return nil
}

	

