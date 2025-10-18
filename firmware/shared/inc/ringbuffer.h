#ifndef RINGBUFFER_H
#define RINGBUFFER_H
#include <stdatomic.h>
/**
 * Provides macros for generating a ringbuffer implementation.
 * This ringbuffer is safe with on a single core, one reader and one writer.
 * Can be used for posting messages from interrupts onto a main thread.
 */

/**
 * Generate a ringbuffer type named <t_name>,
 * containing <size> elements of type <type>.
 */
#define RINGBUFFER_DEF(type, size, t_name)                                                                                                                                                             \
    typedef struct _##t_name {                                                                                                                                                                         \
        type volatile data[size];                                                                                                                                                                      \
                                                                                                                                                                                                       \
        volatile atomic_size_t write_ix;                                                                                                                                                               \
        volatile atomic_size_t read_ix;                                                                                                                                                                \
                                                                                                                                                                                                       \
    } t_name;                                                                                                                                                                                          \
                                                                                                                                                                                                       \
    void t_name##_init(t_name* buf);                                                                                                                                                                   \
    size_t t_name##_size(t_name* buf);                                                                                                                                                                 \
    int t_name##_write(t_name* buf, type elem);                                                                                                                                                        \
    int t_name##_read(t_name* buf, type* elem);

/**
 * Generate implementation code for ringbuffer type <t_name>,
 * containing <size> elements of type <type>.
 */
#define RINGBUFFER_IMPL(type, size, t_name)                                                                                                                                                            \
    void t_name##_init(t_name* buf)                                                                                                                                                                    \
    {                                                                                                                                                                                                  \
        buf->write_ix = 0;                                                                                                                                                                             \
        buf->read_ix = 0;                                                                                                                                                                              \
    }                                                                                                                                                                                                  \
                                                                                                                                                                                                       \
    size_t t_name##_size(t_name* buf) { return buf->read_ix - buf->write_ix; }                                                                                                                         \
                                                                                                                                                                                                       \
    int t_name##_write(t_name* buf, type elem)                                                                                                                                                         \
    {                                                                                                                                                                                                  \
        if (t_name##_size(buf) == size) {                                                                                                                                                              \
            return 0;                                                                                                                                                                                  \
        }                                                                                                                                                                                              \
        buf->data[buf->write_ix % size] = elem;                                                                                                                                                        \
        buf->write_ix++;                                                                                                                                                                               \
        return 1;                                                                                                                                                                                      \
    }                                                                                                                                                                                                  \
    int t_name##_read(t_name* buf, type* elem)                                                                                                                                                         \
    {                                                                                                                                                                                                  \
        if (t_name##_size(buf) == 0) {                                                                                                                                                                 \
            return 0;                                                                                                                                                                                  \
        }                                                                                                                                                                                              \
        *elem = buf->data[buf->read_ix % size];                                                                                                                                                        \
        buf->read_ix++;                                                                                                                                                                                \
        return 1;                                                                                                                                                                                      \
    }

#endif
