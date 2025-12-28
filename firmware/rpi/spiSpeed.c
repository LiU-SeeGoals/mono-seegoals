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

#define	TRUE	(1==1)
#define	FALSE	(!TRUE)

#define	SPI_CHAN		0
#define	NUM_TIMES		100
#define	MAX_SIZE		(1024)

static int myFd;


void spiSetup (int speed)
{
  if ((myFd = wiringPiSPISetupMode (SPI_CHAN, speed, 0)) < 0)
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
  int spiFail ;
  unsigned char myData[DATA_SIZE];

  int speed = 2;
  wiringPiSetup () ;

  spiSetup (speed * 1000000) ;
  //spiSetup (10000) ;
  while(1){
	  spiFail = FALSE ;
	  if (wiringPiSPIDataRW (SPI_CHAN, myData, DATA_SIZE) == -1)
	  {
		printf ("SPI failure: %s\n", strerror (errno)) ;
		spiFail = TRUE ;
	  }
	  for (int i = 0; i < DATA_SIZE; i ++)
	  {
		printBits(myData[i]);
	  }

	  printf("\n");
  }
  close (myFd) ;

  return 0 ;
}
