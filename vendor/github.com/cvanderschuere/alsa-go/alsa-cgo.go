package alsa

/*
	//C imports
	#cgo pkg-config: alsa
	#include <asoundlib.h>
	#include <stdio.h>
	#include <stdlib.h>
	#include <errno.h>
	#include <stdint.h>
	#include <libgen.h>

	//Function is much easier to just to do C...would look cleaner in Go
	snd_pcm_t * alsa_open(char *dev, int rate, int channels)
	{
		snd_pcm_hw_params_t *hwp;
		snd_pcm_sw_params_t *swp;
		snd_pcm_t *h;
		int r;
		int dir;
		snd_pcm_uframes_t period_size_min;
		snd_pcm_uframes_t period_size_max;
		snd_pcm_uframes_t buffer_size_min;
		snd_pcm_uframes_t buffer_size_max;
		snd_pcm_uframes_t period_size;
		snd_pcm_uframes_t buffer_size;

		if ((r = snd_pcm_open(&h, dev, SND_PCM_STREAM_PLAYBACK, 0) < 0))
			return NULL;

		hwp = alloca(snd_pcm_hw_params_sizeof());
		memset(hwp, 0, snd_pcm_hw_params_sizeof());
		snd_pcm_hw_params_any(h, hwp);

		snd_pcm_hw_params_set_access(h, hwp, SND_PCM_ACCESS_RW_INTERLEAVED);
		snd_pcm_hw_params_set_format(h, hwp, SND_PCM_FORMAT_S16_LE);
		snd_pcm_hw_params_set_rate(h, hwp, rate, 0);
		snd_pcm_hw_params_set_channels(h, hwp, channels);

		// Configurue period 

		dir = 0;
		snd_pcm_hw_params_get_period_size_min(hwp, &period_size_min, &dir);
		dir = 0;
		snd_pcm_hw_params_get_period_size_max(hwp, &period_size_max, &dir);

		period_size = 1024;

		dir = 0;
		r = snd_pcm_hw_params_set_period_size_near(h, hwp, &period_size, &dir);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to set period size %lu (%s)\n",
			        period_size, snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		dir = 0;
		r = snd_pcm_hw_params_get_period_size(hwp, &period_size, &dir);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to get period size (%s)\n",
			        snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		//Configurue buffer size

		snd_pcm_hw_params_get_buffer_size_min(hwp, &buffer_size_min);
		snd_pcm_hw_params_get_buffer_size_max(hwp, &buffer_size_max);
		buffer_size = period_size * 4;

		dir = 0;
		r = snd_pcm_hw_params_set_buffer_size_near(h, hwp, &buffer_size);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to set buffer size %lu (%s)\n",
			        buffer_size, snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		r = snd_pcm_hw_params_get_buffer_size(hwp, &buffer_size);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to get buffer size (%s)\n",
			        snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		//write the hw params
		r = snd_pcm_hw_params(h, hwp);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to configure hardware parameters (%s)\n",
			        snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		//Software parameters

		swp = alloca(snd_pcm_sw_params_sizeof());
		memset(hwp, 0, snd_pcm_sw_params_sizeof());
		snd_pcm_sw_params_current(h, swp);

		r = snd_pcm_sw_params_set_avail_min(h, swp, period_size);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to configure wakeup threshold (%s)\n",
			        snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}
		

		r = snd_pcm_sw_params_set_start_threshold(h, swp, 0);

		if (r < 0) {
			fprintf(stderr, "audio: Unable to configure start threshold (%s)\n",
			        snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		r = snd_pcm_sw_params(h, swp);

		if (r < 0) {
			fprintf(stderr, "audio: Cannot set soft parameters (%s)\n",
			snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		r = snd_pcm_prepare(h);
		if (r < 0) {
			fprintf(stderr, "audio: Cannot prepare audio for playback (%s)\n",
			snd_strerror(r));
			snd_pcm_close(h);
			return NULL;
		}

		return h;
	}	
*/
import "C"
import "unsafe"

type alsa_device struct{
	pcm *C.snd_pcm_t
	channels int
	rate int
	numBytes int
}

//For future use
type SampleType uint
const(
	_ SampleType = iota // Leave 0 blank for error checking
	INT16_TYPE
	UINT16_TYPE
	INT32_TYPE
	UINT32_TYPE
	INT64_TYPE
	UINT64_TYPE
	FLOAT_TYPE
	DOUBLE_TYPE
)

func alsa_open(name string, channels,rate int)(* C.snd_pcm_t, error){
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))	

	device := C.alsa_open(cName,C.int(rate),C.int(channels))
		
	return device,nil
}

func alsa_write(device *alsa_device, data AudioData){
	c := C.snd_pcm_wait(device.pcm, 1000);

	var error2 C.snd_pcm_sframes_t
	if c >= 0{
		error2 = C.snd_pcm_avail_update(device.pcm)
	}

	if error2 == -C.EPIPE{
		C.snd_pcm_prepare(device.pcm)
	}	
	
	bytesPerFrame := device.channels * device.numBytes
	numFrames := C.snd_pcm_uframes_t(len(data)/bytesPerFrame)		
	numWritten := C.snd_pcm_writei(device.pcm, unsafe.Pointer(&data[0]), numFrames)
	
	switch{
	case numWritten<=0:	
		log.Warn("Alsa:Error writei");
		C.snd_pcm_recover(device.pcm,C.int(numWritten),0)
	case numWritten>0&&numWritten!=C.snd_pcm_sframes_t(numFrames):
		log.Error("Alsa:wrote less than num frames")
	}
}

func alsa_play(device *alsa_device){
	C.snd_pcm_pause(device.pcm,C.int(1))
}
func alsa_pause(device *alsa_device){
	C.snd_pcm_pause(device.pcm,C.int(0))
}

func alsa_close(device *C.snd_pcm_t){
	if device != nil{
		C.snd_pcm_close(device)
	}
}
