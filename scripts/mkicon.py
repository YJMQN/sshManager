import struct
import zlib

def make_ico_png(size, r, g, b):
    width = height = size
    pixels = []
    for y in range(height):
        row = []
        for x in range(width):
            margin = 2
            in_shape = (x >= margin and x < width - margin and
                        y >= margin and y < height - margin)
            
            is_monitor = False
            if in_shape:
                if (y >= margin + 2 and y < height - margin - 2 and
                        x >= margin + 2 and x < width - margin - 2):
                    is_monitor = True

            # Power button
            is_power = False
            pcx, pcy = width // 2, height // 2 + 2
            pdist = ((x - pcx) ** 2 + (y - pcy) ** 2) ** 0.5
            if abs(pdist - 5) < 1.5 and y < pcy:
                is_power = True

            # Terminal prompt
            is_prompt = False
            if (y >= height // 2 - 2 and y < height // 2 + 2 and
                    x >= width // 2 - 6 and x < width // 2 + 6 and
                    in_shape):
                is_prompt = True

            if is_power:
                pr, pg, pb = 0, 180, 80
            elif is_prompt:
                pr, pg, pb = 0, 200, 100
            elif is_monitor:
                dx = x - width // 2
                dy = y - height // 2
                d = (dx * dx + dy * dy) ** 0.5
                factor = max(0.6, 1.0 - d / (width // 2) * 0.4)
                pr = int(r * factor)
                pg = int(g * factor)
                pb = int(b * factor)
            else:
                pr, pg, pb = 0, 0, 0

            a = 255
            if not is_monitor and not is_power and not is_prompt:
                a = 0

            pixels.append((pr, pg, pb, a))

    def make_chunk(chunk_type, data):
        c = chunk_type + data
        crc = struct.pack('>I', zlib.crc32(c) & 0xffffffff)
        return struct.pack('>I', len(data)) + c + crc

    sig = b'\x89PNG\r\n\x1a\n'
    ihdr = struct.pack('>IIBBBBB', width, height, 8, 6, 0, 0, 0)

    raw = b''
    for y in range(height):
        raw += b'\x00'
        for x in range(width):
            idx = y * width + x
            pr, pg, pb, pa = pixels[idx]
            raw += struct.pack('BBBB', pr, pg, pb, pa)

    compressed = zlib.compress(raw)
    iend = b''

    return sig + make_chunk(b'IHDR', ihdr) + make_chunk(b'IDAT', compressed) + make_chunk(b'IEND', iend)


def make_ico(sizes=None):
    if sizes is None:
        sizes = [16, 32, 48]
    images = []
    for size in sizes:
        png_data = make_ico_png(size, 30, 100, 200)
        images.append((size, size, png_data))

    count = len(images)
    header = struct.pack('<HHH', 0, 1, count)

    offset = 6 + 16 * count
    entries = b''
    for size, _, data in images:
        w = size if size < 256 else 0
        h = size if size < 256 else 0
        size_bytes = len(data)
        entries += struct.pack('<BBBBHHII', w, h, 0, 0, 1, 32, size_bytes, offset)
        offset += size_bytes

    data_block = b''.join(d for _, _, d in images)
    return header + entries + data_block


ico_data = make_ico([16, 32, 48])
with open('/root/.qwenpaw/workspaces/HQWjoj/ssh_manager_go/app.ico', 'wb') as f:
    f.write(ico_data)
print("Icon created: %d bytes, %d images" % (len(ico_data), 3))
