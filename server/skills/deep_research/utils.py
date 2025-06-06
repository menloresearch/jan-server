import xml.etree.ElementTree as ET


def parse_xml_tags(xml_string):
    """Parse XML string and extract text from specific tags"""
    try:
        root = ET.fromstring(f"<root>{xml_string}</root>")

        results = {}

        for element in root:
            tag_name = element.tag
            tag_text = element.text if element.text else ""
            results[tag_name] = tag_text.strip()

        return results

    except ET.ParseError as e:
        print(f"XML parsing error: {e}")
        return {}
