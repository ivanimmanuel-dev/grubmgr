import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('build_deb', Path(__file__).with_name('build-deb.py'))
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class SourceArchiveTest(unittest.TestCase):
    def test_archive_uses_source_timestamp(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root/'internal/cli/cli.go'
            source.parent.mkdir(parents=True)
            source.write_text('package cli\n')
            os.utime(source, (1000000000, 1000000000))
            with patch.object(builder, 'ROOT', root):
                self.assertEqual(builder.source_epoch({}), 1000000000)

    def test_explicit_timestamp_works_without_git_or_source_metadata(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(builder, 'ROOT', Path(directory)):
                with patch.object(builder.subprocess, 'check_output', side_effect=AssertionError('Git must not run')):
                    self.assertEqual(builder.source_epoch({'SOURCE_DATE_EPOCH': '0'}), 0)

    def test_invalid_timestamp_is_rejected(self):
        for value in ['', '-1', '1.5', 'yesterday']:
            with self.subTest(value=value), self.assertRaisesRegex(SystemExit, 'nonnegative integer'):
                builder.source_epoch({'SOURCE_DATE_EPOCH': value})


if __name__ == '__main__':
    unittest.main()
