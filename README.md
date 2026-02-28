# leonardo-cli

`leonardo-cli` is a simple command‑line tool written in Go that wraps parts of the [Leonardo.Ai API](https://docs.leonardo.ai/).  It allows you to kick off image generation jobs and poll their status from your own scripts or terminals.  Under the hood the code is organised following [hexagonal/clean architecture](https://en.wikipedia.org/wiki/Hexagonal_architecture) so that core logic and domain models are isolated from external dependencies such as HTTP clients or the CLI itself.

* `create` — Start a new text‑to‑image generation.  You can specify your prompt and optional parameters like model ID, image dimensions, number of images and more.
* `status` — Check the progress of a previously started generation using its ID.  This command reports the status and prints any available image URLs once the job is complete.
* `download` — Download completed generation images locally and write a JSON sidecar metadata file next to each image.
* `inspect` — Read and pretty-print a sidecar metadata JSON file.

## Requirements

* Go 1.20 or later to build from source.
* A Leonardo.Ai API key with sufficient credits.  Set the key in your environment as `LEONARDO_API_KEY` before running the CLI.  You can obtain a key from your account’s **API Access** page on Leonardo.Ai.

## Installation

Clone the repository and build the binary:

```sh
git clone <repo-url> leonardo-cli
cd leonardo-cli
go build -o leonardo ./cmd/leonardo
```

Make sure your `LEONARDO_API_KEY` environment variable is exported:

```sh
export LEONARDO_API_KEY="your‑api‑key-here"
```

## Usage

Run `leonardo` without any arguments to see the available commands:

```sh
./leonardo
```

### Create a generation

The `create` command submits a new image generation request.  A prompt is required.  Optional flags let you control the model, resolution and other parameters.  For example:

```sh
./leonardo create \
  --prompt "A serene watercolor painting of a mountain lake at sunrise" \
  --model-id 7b592283-e8a7-4c5a-9ba6-d18c31f258b9 \
  --width 1920 \
  --height 1080 \
  --num-images 4 \
  --style-uuid 111dc692-d470-4eec-b791-3475abac4c46 \
  --contrast 3.5 \
  --alchemy=false \
  --ultra=false
```

If the call is successful, the CLI prints the returned `generationId` along with the full JSON response.  The generation ID can be used to poll for status.

In the [Quick Start Guide](https://docs.leonardo.ai/docs/getting-started), Leonardo explains that after submitting a generation you receive an identifier (often called `generationId`) that is used in subsequent calls【202409399148263†L150-L176】.

### Check generation status

Use the `status` command with the generation ID to check if your images are ready:

```sh
./leonardo status --id 123456-0987-aaaa-bbbb-01010101010
```

The CLI will query `GET /api/rest/v1/generations/{id}`.  It attempts to extract the `status` and any image URLs from the response.  If the status is `PENDING`, no images will be returned.  According to Leonardo’s API FAQs, the `generated_images` array remains empty until the job is complete【271928005095238†L183-L201】.  Once the status is `COMPLETE`, the response contains URLs to the generated images.

The full JSON response is printed for completeness.

### Download images and sidecar metadata

When you download completed images, the CLI writes each image and a matching sidecar JSON file:

```sh
./leonardo download --id 12345678-90ab-cdef-1234-567890abcdef --output-dir ./out
```

For an image like `./out/<generation-id>_1.png`, a sidecar `./out/<generation-id>_1.png.json` is also written.  The sidecar includes generation metadata such as generation ID, image index, timestamp, and generation parameters parsed from the API payload.

Because metadata is saved from the decoded generation payload, newly added generation parameters are automatically preserved without requiring README or code updates for each new field.

### Inspect a sidecar JSON file

Use the `inspect` command to view sidecar metadata:

```sh
./leonardo inspect --file ./out/12345678-90ab-cdef-1234-567890abcdef_1.png.json
```

## Architecture overview

The project is split into layers to make the code easier to extend and test:

* **Domain (`internal/domain`)**: Contains simple structs representing requests and responses (`GenerationRequest`, `GenerationResponse` and `GenerationStatus`).  These types model the core concepts of the application without knowledge of external libraries.
* **Ports (`internal/ports`)**: Defines the `LeonardoClient` interface that describes the operations needed to interact with the Leonardo service.  Any adapter (HTTP, mock, etc.) implementing this interface can be plugged into the service.
* **Provider (`internal/provider`)**: Provides a concrete implementation of the `LeonardoClient` that talks to the Leonardo REST API over HTTP.  The `APIClient` in this layer builds requests, handles authentication and parses responses.
* **Service (`internal/service`)**: Implements the application logic by depending on the `LeonardoClient` port.  The `GenerationService` exposes methods to create a generation and check its status.  Because it relies on an interface, the service can be tested with a mock client.
* **CLI (`cmd/leonardo`)**: The entrypoint that parses command‑line flags and calls into the service layer.  It does not know about HTTP details; those are handled by the provider.

This structure keeps the domain and business logic decoupled from I/O so that the tool can be adapted for other interfaces (for example, a GUI or web server) by providing alternative implementations of the `LeonardoClient` port.

## Notes

* Only the most common parameters are exposed as flags.  Refer to the official documentation for advanced options such as `guidance_scale`, `init_image_id` and ControlNet parameters.
* The CLI does not implement any retry logic.  For long‑running jobs you may need to poll repeatedly until the status changes to `COMPLETE`【271928005095238†L183-L201】.
* Ensure your API key has sufficient credits.  The API will return an error if your credit balance is low.

## License

This project is provided under the MIT License.  See `LICENSE` for details.
