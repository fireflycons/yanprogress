# yanprogress

Yet ANother Progress Bar :smile:

Dead simple no-frills progress bar that writes by default to stderr.

There are two types
* Bounded - where the total number of iterations to process is known in advance. This will produce a bar output like this

    ```
    [==============>               ]  50% (19.7 it/s)
    ```
* Unbounded - where the total number of iterations to process is not known in advance. This will produce a spinner output like this

    ```
    ⠋ (19.8 it/s)
    ```

If the terminal is not a tty (destination is a file or redirected), then it will produce sequential output like this, without the percentage if unbounded.

```
  9% (18.0 it/s)
 19% (19.0 it/s)
 28% (19.3 it/s)
 39% (19.5 it/s)
 49% (19.6 it/s)
 ```

You may optionally add a status to say what it is doing. This can be called at any point to update the status line, in which case it looks like this

```
Frobbing the turnips
[==============>               ]  50% (19.7 it/s)
```

## Examples

Bounded progress bar. To use unbounded set the first argument to zero

```go
bar := NewProgressBar(100, 500*time.Millisecond)
bar.SetStatus("Downloading File")
bar.Start()

// Simulate work
for i := 0; i <= 100; i++ {
    time.Sleep(50 * time.Millisecond)
    bar.Inc()

    if i == 50 {
        bar.SetStatus("Half way there!")
    }
}
bar.Complete()

```

Write to stdout

```go
bar := NewProgressBar(100, 500*time.Millisecond, WithWriter(os.Stdout))
```
