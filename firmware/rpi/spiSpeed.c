#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>

#include <wiringPi.h>
#include <wiringPiSPI.h>
#include "imu.pb.h"
#include "pb_encode.h"
#include "pb_decode.h"

#define	SPI_CHAN		0
#define	SPI_MODE		1
#define DATA_SIZE       257

static int myFd;

bool spiSetup (int speed)
{

    if ((myFd = wiringPiSPISetupMode (SPI_CHAN, speed, SPI_MODE)) < 0)
    {
        fprintf (stderr, "Can't open the SPI bus: %s\n", strerror (errno)) ;
        // exit (EXIT_FAILURE) ;
        return false;
    }
    return true;
}

int getDataSize(){
    return DATA_SIZE;
}

void printBits(unsigned char x){
	unsigned char num = 8;

	for(int i = 1; i <= num; i++)
	    printf("%d", (x >> (num - i)) & 1);
    printf(" ");
}

void spiOpen()
{
  wiringPiSetup();
  int speed = 1;
  spiSetup (speed * 850000) ;
}

void spiClose()
{
  close(myFd) ;
}

bool spiRead(uint8_t* out)
{
  uint8_t spiData[DATA_SIZE];

  if (wiringPiSPIDataRW (SPI_CHAN, spiData, DATA_SIZE) == -1)
  {
    printf ("SPI failure: %s\n", strerror (errno)) ;
  }
  uint8_t msg_length = spiData[0];

  if (msg_length < 1)
  {
      return false;
  }

  for (int i = 0; i < msg_length; i ++)
  {
    // printBits(spiData[i]);
    // printf("%d ", spiData[i]);
  }

  for (int i = 0; i < DATA_SIZE; i ++)
  {
      out[i] = spiData[i];
  }

  return true;
}

int main (void)
{
    spiOpen();
    uint8_t out[DATA_SIZE];
    int time_stamp_dt = 0;
    int numFailed = 0;
    int numSuccess;
    while(1)
    {
        bool status = spiRead(out);
        ImuSample msg = ImuSample_init_zero;
        uint8_t msg_length = out[0];

        pb_istream_t stream = pb_istream_from_buffer(out + 1, msg_length);

        if (!pb_decode(&stream, ImuSample_fields, &msg))
        {
            // printf("Failed decode\n");
            numFailed++;
            continue;
        }

        static float prev_timestamp = 0;
        time_stamp_dt = msg.imu_ts - prev_timestamp;
        if (time_stamp_dt == 0 || msg.imu_ts < prev_timestamp)
        {
          numFailed++;
          continue;
        }
        printf("%d %f %d %f %d %f\n", msg_length, msg.imu_z, msg.imu_ts,msg.state_z, msg.state_ts, 1000.f/(float)time_stamp_dt);
        prev_timestamp=msg.imu_ts;
        numSuccess++;
    }
    spiClose();
  return 0 ;
}
