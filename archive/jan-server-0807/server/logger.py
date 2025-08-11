import logging
import os


logging.getLogger("httpx").setLevel(logging.WARNING)


class Formatter(logging.Formatter):
    COLORS = {
        "INFO": "\033[92m",  # Green
        "WARNING": "\033[93m",  # Yellow
        "ERROR": "\033[91m",  # Red
        "CRITICAL": "\033[95m",  # Magenta
        "DEBUG": "\033[94m",  # Blue
        "RESET": "\033[0m",
    }

    def format(self, record):
        # Convert to integer milliseconds (like JavaScript Date.now())
        epoch_ms = int(record.created * 1000)

        color = self.COLORS.get(record.levelname, self.COLORS["RESET"])
        reset = self.COLORS["RESET"]

        # Format with colored level name and epoch timestamp
        return f"{color}[{record.levelname}]{reset} ({epoch_ms}): {record.getMessage()}"


handler = logging.StreamHandler()
handler.setFormatter(Formatter())

logger = logging.getLogger(__name__)
log_level = os.getenv("LOG_LEVEL", "INFO").upper()
level = getattr(logging, log_level, logging.INFO)
logger.setLevel(level)

logger.addHandler(handler)
logger.propagate = False
