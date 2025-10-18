/* USER CODE BEGIN Header */
/**
 ******************************************************************************
 * @file    app_netxduo.c
 * @author  MCD Application Team
 * @brief   NetXDuo applicative file
 ******************************************************************************
 * @attention
 *
 * Copyright (c) 2023 STMicroelectronics.
 * All rights reserved.
 *
 * This software is licensed under terms that can be found in the LICENSE file
 * in the root directory of this software component.
 * If no LICENSE file comes with this software, it is provided AS-IS.
 *
 ******************************************************************************
 */
/* USER CODE END Header */

/* Includes ------------------------------------------------------------------*/
#include "app_netxduo.h"

/* Private includes ----------------------------------------------------------*/
#include "nxd_dhcp_client.h"
/* USER CODE BEGIN Includes */
#include "main.h"
#include <com.h>
#include <log.h>
/* USER CODE END Includes */

/* Private typedef -----------------------------------------------------------*/
/* USER CODE BEGIN PTD */

/* USER CODE END PTD */

/* Private define ------------------------------------------------------------*/
/* USER CODE BEGIN PD */

/* USER CODE END PD */

/* Private macro -------------------------------------------------------------*/
/* USER CODE BEGIN PM */

/* USER CODE END PM */

/* Private variables ---------------------------------------------------------*/
TX_THREAD NxAppThread;
NX_PACKET_POOL NxAppPool;
NX_IP NetXDuoEthIpInstance;
TX_SEMAPHORE DHCPSemaphore;
NX_DHCP DHCPClient;
/* USER CODE BEGIN PV */
static LOG_Module internal_log_mod;
ULONG IPAddress;
ULONG Netmask;
TX_THREAD NxUDPThread;
TX_THREAD NxLinkThread;
NX_UDP_SOCKET visionSocket;
NX_UDP_SOCKET controllerSocket;
ULONG VISION_PORT = 10006;
ULONG CONTROLLER_PORT = 6001;
ULONG QUEUE_MAX_SIZE = 512;
/* USER CODE END PV */

/* Private function prototypes -----------------------------------------------*/
static VOID nx_app_thread_entry(ULONG thread_input);
static VOID ip_address_change_notify_callback(NX_IP* ip_instance, VOID* ptr);
/* USER CODE BEGIN PFP */
static VOID nx_link_thread_entry(ULONG thread_input);
static VOID nx_udp_thread_entry(ULONG thread_input);
static VOID udp_socket_receive_vision(NX_UDP_SOCKET* socket_ptr);
static VOID udp_socket_receive_controller(NX_UDP_SOCKET* socket_ptr);
/* USER CODE END PFP */

/**
 * @brief  Application NetXDuo Initialization.
 * @param memory_ptr: memory pointer
 * @retval int
 */
UINT MX_NetXDuo_Init(VOID* memory_ptr)
{
    UINT ret = NX_SUCCESS;
    TX_BYTE_POOL* byte_pool = (TX_BYTE_POOL*)memory_ptr;

    /* USER CODE BEGIN App_NetXDuo_MEM_POOL */
    (void)byte_pool;
    /* USER CODE END App_NetXDuo_MEM_POOL */
    /* USER CODE BEGIN 0 */
    LOG_InitModule(&internal_log_mod, "NX", LOG_LEVEL_DEBUG, 0);
    /* USER CODE END 0 */

    /* Initialize the NetXDuo system. */
    CHAR* pointer;
    nx_system_initialize();

    /* Allocate the memory for packet_pool.  */
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, NX_APP_PACKET_POOL_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }

    /* Create the Packet pool to be used for packet allocation,
     * If extra NX_PACKET are to be used the NX_APP_PACKET_POOL_SIZE should be increased
     */
    ret = nx_packet_pool_create(&NxAppPool, "NetXDuo App Pool", DEFAULT_PAYLOAD_SIZE, pointer, NX_APP_PACKET_POOL_SIZE);

    if (ret != NX_SUCCESS) {
        return NX_POOL_ERROR;
    }

    /* Allocate the memory for Ip_Instance */
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, Nx_IP_INSTANCE_THREAD_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }

    /* Create the main NX_IP instance */
    ret = nx_ip_create(&NetXDuoEthIpInstance, "NetX Ip instance", NX_APP_DEFAULT_IP_ADDRESS, NX_APP_DEFAULT_NET_MASK, &NxAppPool, nx_stm32_eth_driver, pointer, Nx_IP_INSTANCE_THREAD_SIZE,
                       NX_APP_INSTANCE_PRIORITY);

    if (ret != NX_SUCCESS) {
        return NX_NOT_SUCCESSFUL;
    }

    /* Allocate the memory for ARP */
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, DEFAULT_ARP_CACHE_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }

    /* Enable the ARP protocol and provide the ARP cache size for the IP instance */

    /* USER CODE BEGIN ARP_Protocol_Initialization */

    /* USER CODE END ARP_Protocol_Initialization */

    ret = nx_arp_enable(&NetXDuoEthIpInstance, (VOID*)pointer, DEFAULT_ARP_CACHE_SIZE);

    if (ret != NX_SUCCESS) {
        return NX_NOT_SUCCESSFUL;
    }

    /* Enable the ICMP */

    /* USER CODE BEGIN ICMP_Protocol_Initialization */

    /* USER CODE END ICMP_Protocol_Initialization */

    ret = nx_icmp_enable(&NetXDuoEthIpInstance);

    if (ret != NX_SUCCESS) {
        return NX_NOT_SUCCESSFUL;
    }

    /* Enable TCP Protocol */

    /* USER CODE BEGIN TCP_Protocol_Initialization */

    /* USER CODE END TCP_Protocol_Initialization */

    ret = nx_tcp_enable(&NetXDuoEthIpInstance);

    if (ret != NX_SUCCESS) {
        return NX_NOT_SUCCESSFUL;
    }

    /* Enable the UDP protocol required for  DHCP communication */

    /* USER CODE BEGIN UDP_Protocol_Initialization */

    /* USER CODE END UDP_Protocol_Initialization */

    ret = nx_udp_enable(&NetXDuoEthIpInstance);

    if (ret != NX_SUCCESS) {
        return NX_NOT_SUCCESSFUL;
    }

    /* Allocate the memory for main thread   */
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, NX_APP_THREAD_STACK_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }

    /* Create the main thread */
    ret = tx_thread_create(&NxAppThread, "NetXDuo App thread", nx_app_thread_entry, 0, pointer, NX_APP_THREAD_STACK_SIZE, NX_APP_THREAD_PRIORITY, NX_APP_THREAD_PRIORITY, TX_NO_TIME_SLICE,
                           TX_AUTO_START);

    if (ret != TX_SUCCESS) {
        return TX_THREAD_ERROR;
    }

    /* Create the DHCP client */

    /* USER CODE BEGIN DHCP_Protocol_Initialization */

    /* USER CODE END DHCP_Protocol_Initialization */

    ret = nx_dhcp_create(&DHCPClient, &NetXDuoEthIpInstance, "DHCP Client");

    if (ret != NX_SUCCESS) {
        return NX_DHCP_ERROR;
    }

    /* set DHCP notification callback  */
    tx_semaphore_create(&DHCPSemaphore, "DHCP Semaphore", 0);

    /* USER CODE BEGIN MX_NetXDuo_Init */

    // Create the link thread
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, NX_APP_THREAD_STACK_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }
    ret = tx_thread_create(&NxLinkThread, "NetXDuo link thread", nx_link_thread_entry, 0, pointer, NX_APP_THREAD_STACK_SIZE, 11, 11, TX_NO_TIME_SLICE, TX_AUTO_START);
    if (ret != TX_SUCCESS) {
        return NX_NOT_ENABLED;
    }

    // Create the UDP thread
    if (tx_byte_allocate(byte_pool, (VOID**)&pointer, NX_APP_THREAD_STACK_SIZE, TX_NO_WAIT) != TX_SUCCESS) {
        return TX_POOL_ERROR;
    }
    ret = tx_thread_create(&NxUDPThread, "NetXDuo UDP thread", nx_udp_thread_entry, 0, pointer, NX_APP_THREAD_STACK_SIZE, NX_APP_THREAD_PRIORITY, NX_APP_THREAD_PRIORITY, TX_NO_TIME_SLICE,
                           TX_DONT_START);
    if (ret != TX_SUCCESS) {
        return NX_NOT_ENABLED;
    }
    /* USER CODE END MX_NetXDuo_Init */

    return ret;
}

/**
 * @brief  ip address change callback.
 * @param ip_instance: NX_IP instance
 * @param ptr: user data
 * @retval none
 */
static VOID ip_address_change_notify_callback(NX_IP* ip_instance, VOID* ptr)
{
    /* USER CODE BEGIN ip_address_change_notify_callback */
    nx_ip_address_get(ip_instance, &IPAddress, &Netmask);
    LOG_INFO("Got IP: %lu.%lu.%lu.%lu \r\n", (IPAddress >> 24) & 0xff, (IPAddress >> 16) & 0xff, (IPAddress >> 8) & 0xff, (IPAddress & 0xff));
    /* USER CODE END ip_address_change_notify_callback */

    /* release the semaphore as soon as an IP address is available */
    tx_semaphore_put(&DHCPSemaphore);
}

/**
 * @brief  Main thread entry.
 * @param thread_input: ULONG user argument used by the thread entry
 * @retval none
 */
static VOID nx_app_thread_entry(ULONG thread_input)
{
    /* USER CODE BEGIN Nx_App_Thread_Entry 0 */

    /* USER CODE END Nx_App_Thread_Entry 0 */

    UINT ret = NX_SUCCESS;

    /* USER CODE BEGIN Nx_App_Thread_Entry 1 */

    /* USER CODE END Nx_App_Thread_Entry 1 */

    /* register the IP address change callback */
    ret = nx_ip_address_change_notify(&NetXDuoEthIpInstance, ip_address_change_notify_callback, NULL);
    if (ret != NX_SUCCESS) {
        /* USER CODE BEGIN IP address change callback error */
        Error_Handler();
        /* USER CODE END IP address change callback error */
    }

    /* start the DHCP client */
    ret = nx_dhcp_start(&DHCPClient);
    if (ret != NX_SUCCESS) {
        /* USER CODE BEGIN DHCP client start error */
        Error_Handler();
        /* USER CODE END DHCP client start error */
    }

    /* wait until an IP address is ready */
    if (tx_semaphore_get(&DHCPSemaphore, NX_APP_DEFAULT_TIMEOUT) != TX_SUCCESS) {
        /* USER CODE BEGIN DHCPSemaphore get error */
        while (tx_semaphore_get(&DHCPSemaphore, NX_APP_DEFAULT_TIMEOUT) != TX_SUCCESS) {
        }
        /* USER CODE END DHCPSemaphore get error */
    }

    /* USER CODE BEGIN Nx_App_Thread_Entry 2 */
    tx_thread_resume(&NxUDPThread);
    /* USER CODE END Nx_App_Thread_Entry 2 */
}

/* USER CODE BEGIN 1 */

static VOID nx_link_thread_entry(ULONG thread_input)
{
    ULONG status;
    UINT ret = NX_SUCCESS;
    UINT linkdown = -1;

    for (;;) {
        ret = nx_ip_interface_status_check(&NetXDuoEthIpInstance, 0, NX_IP_LINK_ENABLED, &status, 10);

        if (ret != NX_SUCCESS) {
            if (linkdown != 1) {
                linkdown = 1;
                LOG_INFO("Link down...\r\n");
                HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_SET);
                HAL_GPIO_WritePin(LED_GREEN_GPIO_Port, LED_GREEN_Pin, GPIO_PIN_RESET);
            }

            // Indicate on LED
            HAL_GPIO_TogglePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin);
            HAL_Delay(500);
            HAL_GPIO_TogglePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin);
        } else {
            if (linkdown == 1) {
                linkdown = 0;
                LOG_INFO("Link up...\r\n");

                ret = nx_ip_interface_status_check(&NetXDuoEthIpInstance, 0, NX_IP_ADDRESS_RESOLVED, &status, 10);

                if (ret == NX_SUCCESS) {
                    LOG_INFO("IP resolved...\r\n");
                    HAL_GPIO_WritePin(LED_GREEN_GPIO_Port, LED_GREEN_Pin, GPIO_PIN_SET);
                    HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_RESET);
                } else {
                    LOG_INFO("IP not resolved...\r\n");
                    HAL_GPIO_WritePin(LED_GREEN_GPIO_Port, LED_GREEN_Pin, GPIO_PIN_RESET);
                    HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_SET);
                    nx_ip_driver_direct_command(&NetXDuoEthIpInstance, NX_LINK_ENABLE, &status);
                    nx_dhcp_stop(&DHCPClient);
                    nx_dhcp_start(&DHCPClient);
                }
            } else {
                linkdown = 0;
                HAL_GPIO_WritePin(LED_GREEN_GPIO_Port, LED_GREEN_Pin, GPIO_PIN_SET);
                HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_RESET);
            }
        }

        tx_thread_sleep(NX_LINK_CHECK_PERIOD);
    }
}

static VOID nx_udp_thread_entry(ULONG thread_input)
{
    UINT ret = NX_SUCCESS;

    ret = nx_udp_socket_create(&NetXDuoEthIpInstance, &visionSocket, "UDP Client Socket", NX_IP_NORMAL, NX_FRAGMENT_OKAY, NX_IP_TIME_TO_LIVE, QUEUE_MAX_SIZE);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }

    ret = nx_udp_socket_bind(&visionSocket, VISION_PORT, TX_WAIT_FOREVER);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }

    ret = nx_udp_socket_receive_notify(&visionSocket, udp_socket_receive_vision);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }

    LOG_INFO("Waiting for Proto packets on port %lu...\r\n", VISION_PORT);

    ret = nx_udp_socket_create(&NetXDuoEthIpInstance, &controllerSocket, "UDP Client Socket", NX_IP_NORMAL, NX_FRAGMENT_OKAY, NX_IP_TIME_TO_LIVE, QUEUE_MAX_SIZE);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }

    ret = nx_udp_socket_bind(&controllerSocket, CONTROLLER_PORT, TX_WAIT_FOREVER);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }

    ret = nx_udp_socket_receive_notify(&controllerSocket, udp_socket_receive_controller);
    if (ret != NX_SUCCESS) {
        Error_Handler();
    }
    LOG_INFO("Waiting for robot actions on port %lu...\r\n", CONTROLLER_PORT);

    tx_thread_relinquish();
}

static VOID udp_socket_receive_vision(NX_UDP_SOCKET* socket_ptr)
{
    UINT ret = NX_SUCCESS;
    NX_PACKET* data_packet;

    ret = nx_udp_socket_receive(socket_ptr, &data_packet, NX_APP_DEFAULT_TIMEOUT);
    if (ret == NX_SUCCESS) {
        COM_ParsePacket(data_packet, SSL_WRAPPER);
        nx_packet_release(data_packet);
    }
}

static VOID udp_socket_receive_controller(NX_UDP_SOCKET* socket_ptr)
{
    UINT ret = NX_SUCCESS;
    NX_PACKET* data_packet;

    ret = nx_udp_socket_receive(socket_ptr, &data_packet, NX_APP_DEFAULT_TIMEOUT);
    if (ret == NX_SUCCESS) {
        COM_ParsePacket(data_packet, ROBOT_COMMAND);
        nx_packet_release(data_packet);
    }
}

/* USER CODE END 1 */
