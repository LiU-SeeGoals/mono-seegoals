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

#define	TRUE	(1==1)
#define	FALSE	(!TRUE)

#define	SPI_CHAN		0
#define	SPI_MODE		1
#define	NUM_TIMES		100
#define	MAX_SIZE		(1024)

static int myFd;


void spiSetup (int speed)
{
  if ((myFd = wiringPiSPISetupMode (SPI_CHAN, speed, SPI_MODE)) < 0)
  {
    fprintf (stderr, "Can't open the SPI bus: %s\n", strerror (errno)) ;
    exit (EXIT_FAILURE) ;
  }
}

void printBits(unsigned char x){
	unsigned char num = 8;

	for(int i = 1; i <= num; i++)
	    printf("%d", (x >> (num - i)) & 1);
    printf(" ");
}
#define DATA_SIZE 64

int main (void)
{
  pb_byte_t myData[DATA_SIZE];

  int speed = 4;
  wiringPiSetup () ;



  spiSetup (speed * 500000) ;
  //spiSetup (speed * 1000) ;
  while(1){

	  if (wiringPiSPIDataRW (SPI_CHAN, myData, DATA_SIZE) == -1)
	  {
		printf ("SPI failure: %s\n", strerror (errno)) ;
	  }
	  uint8_t msg_length = myData[0];
	  pb_istream_t stream = pb_istream_from_buffer(myData + 1, msg_length);
	  ImuSample msg = ImuSample_init_zero;
	 
	  if (!pb_decode(&stream, ImuSample_fields, &msg))
	  {
		  printf("Failed decode\n");
	  }
	  printf("%d %f %f %f\n", msg_length, msg.x, msg.y, msg.z);
	  for (int i = 0; i < msg_length; i ++)
	  {
	        // printBits(myData[i]);
	        printf("%d ", myData[i]);
	  }

	  printf("\n===================\n");
  }
  close (myFd) ;

  return 0 ;
}
